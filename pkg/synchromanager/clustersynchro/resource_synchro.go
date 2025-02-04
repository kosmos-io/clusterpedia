package clustersynchro

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/atomic"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	genericstorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"

	clusterv1alpha2 "github.com/clusterpedia-io/api/cluster/v1alpha2"
	kubestatemetrics "github.com/clusterpedia-io/clusterpedia/pkg/kube_state_metrics"
	"github.com/clusterpedia-io/clusterpedia/pkg/storage"
	"github.com/clusterpedia-io/clusterpedia/pkg/synchromanager/clustersynchro/informer"
	"github.com/clusterpedia-io/clusterpedia/pkg/synchromanager/clustersynchro/queue"
	"github.com/clusterpedia-io/clusterpedia/pkg/synchromanager/features"
	"github.com/clusterpedia-io/clusterpedia/pkg/utils"
	clusterpediafeature "github.com/clusterpedia-io/clusterpedia/pkg/utils/feature"
	"github.com/clusterpedia-io/clusterpedia/pkg/watcher/middleware"
)

type ResourceSynchroConfig struct {
	schema.GroupVersionResource
	Kind string

	cache.ListerWatcher
	runtime.ObjectConvertor
	storage.ResourceStorage

	*kubestatemetrics.MetricsStore

	ResourceVersions    map[string]interface{}
	PageSizeForInformer int64
}

func (c ResourceSynchroConfig) GroupVersionKind() schema.GroupVersionKind {
	return c.GroupVersionResource.GroupVersion().WithKind(c.Kind)
}

type ResourceSynchro struct {
	cluster string

	example         runtime.Object
	syncResource    schema.GroupVersionResource
	storageResource schema.GroupVersionResource

	pageSize          int64
	listerWatcher     cache.ListerWatcher
	metricsExtraStore informer.ExtraStore
	metricsWriter     *metricsstore.MetricsWriter

	queue   queue.EventQueue
	cache   *informer.ResourceVersionStorage
	rvs     map[string]interface{}
	rvsLock sync.Mutex

	memoryVersion schema.GroupVersion
	storage       storage.ResourceStorage
	convertor     runtime.ObjectConvertor

	status atomic.Value // clusterv1alpha2.ClusterResourceSyncCondition

	startlock sync.Mutex
	stopped   chan struct{}

	// TODO(Iceber): Optimize variable names
	isRunnableForStorage *atomic.Bool
	forStorageLock       sync.Mutex
	runnableForStorage   chan struct{}
	stopForStorage       chan struct{}

	closeOnce sync.Once
	ctx       context.Context
	cancel    context.CancelFunc
	closer    chan struct{}
	closed    chan struct{}

	// for debug
	runningStage string
}

func newResourceSynchro(cluster string, config ResourceSynchroConfig) *ResourceSynchro {
	storageConfig := config.ResourceStorage.GetStorageConfig()
	synchro := &ResourceSynchro{
		cluster:         cluster,
		syncResource:    config.GroupVersionResource,
		storageResource: storageConfig.StorageGroupResource.WithVersion(storageConfig.StorageVersion.Version),

		pageSize:      config.PageSizeForInformer,
		listerWatcher: config.ListerWatcher,
		rvs:           config.ResourceVersions,

		// all resources saved to the queue are `runtime.Object`
		queue: queue.NewPressureQueue(cache.DeletionHandlingMetaNamespaceKeyFunc),

		storage:       config.ResourceStorage,
		convertor:     config.ObjectConvertor,
		memoryVersion: storageConfig.MemoryVersion,

		stopped:              make(chan struct{}),
		isRunnableForStorage: atomic.NewBool(true),
		runnableForStorage:   make(chan struct{}),
		stopForStorage:       make(chan struct{}),

		closer: make(chan struct{}),
		closed: make(chan struct{}),
	}
	close(synchro.runnableForStorage)
	synchro.ctx, synchro.cancel = context.WithCancel(context.Background())

	example := &unstructured.Unstructured{}
	example.SetGroupVersionKind(config.GroupVersionKind())
	synchro.example = example

	if config.MetricsStore != nil {
		synchro.metricsExtraStore = config.MetricsStore
		synchro.metricsWriter = metricsstore.NewMetricsWriter(config.MetricsStore.MetricsStore)
	}

	synchro.setStatus(clusterv1alpha2.ResourceSyncStatusPending, "", "")
	return synchro
}

func (synchro *ResourceSynchro) Run(shutdown <-chan struct{}) {
	defer close(synchro.closed)
	go func() {
		select {
		case <-shutdown:
			synchro.Close()
		case <-synchro.closer:
		}
	}()

	// make `synchro.Start` runable
	close(synchro.stopped)

	synchro.runningStage = "running"
	wait.Until(func() {
		synchro.processResources()
	}, time.Second, synchro.closer)
	synchro.runningStage = "processorStop"

	synchro.startlock.Lock()
	synchro.runningStage = "waitStop"
	<-synchro.stopped
	synchro.startlock.Unlock()

	synchro.setStatus(clusterv1alpha2.ResourceSyncStatusStop, "", "")
	synchro.runningStage = "shutdown"
}

func (synchro *ResourceSynchro) Close() <-chan struct{} {
	synchro.closeOnce.Do(func() {
		close(synchro.closer)
		synchro.queue.Close()
		synchro.cancel()
	})
	return synchro.closed
}

func (synchro *ResourceSynchro) Start(stopCh <-chan struct{}) {
	synchro.startlock.Lock()
	stopped := synchro.stopped // avoid race
	synchro.startlock.Unlock()
	for {
		select {
		case <-stopCh:
			return
		case <-synchro.closer:
			return
		case <-stopped:
		}

		var dorun bool
		func() {
			synchro.startlock.Lock()
			defer synchro.startlock.Unlock()

			select {
			case <-stopCh:
				stopped = nil
				return
			case <-synchro.closer:
				stopped = nil
				return
			default:
			}

			select {
			case <-synchro.stopped:
				dorun = true
				synchro.stopped = make(chan struct{})
			default:
			}

			stopped = synchro.stopped
		}()

		if dorun {
			break
		}
	}

	defer close(synchro.stopped)
	for {
		synchro.forStorageLock.Lock()
		runnableForStorage, stopForStorage := synchro.runnableForStorage, synchro.stopForStorage
		synchro.forStorageLock.Unlock()

		select {
		case <-stopCh:
			synchro.setStatus(clusterv1alpha2.ResourceSyncStatusStop, "Pause", "")
			return
		case <-synchro.closer:
			return
		case <-runnableForStorage:
		}

		select {
		case <-stopCh:
			synchro.setStatus(clusterv1alpha2.ResourceSyncStatusStop, "Pause", "")
			return
		case <-synchro.closer:
			return
		case <-stopForStorage:
			// stopForStorage is closed, storage is not runnable,
			// continue to get `runnableForStorage` and `stopForStorage`
			continue
		default:
		}

		informerStopCh := make(chan struct{})
		go func() {
			select {
			case <-stopCh:
			case <-synchro.closer:
			case <-stopForStorage:
			}
			close(informerStopCh)
		}()

		synchro.rvsLock.Lock()
		if synchro.cache == nil {
			rvs := make(map[string]interface{}, len(synchro.rvs))
			for r, v := range synchro.rvs {
				rvs[r] = v
			}
			synchro.cache = informer.NewResourceVersionStorage()
			synchro.rvsLock.Unlock()

			_ = synchro.cache.Replace(rvs)
		} else {
			synchro.rvsLock.Unlock()
		}

		config := informer.InformerConfig{
			ListerWatcher:     synchro.listerWatcher,
			Storage:           synchro.cache,
			ExampleObject:     synchro.example,
			Handler:           synchro,
			ErrorHandler:      synchro.ErrorHandler,
			ExtraStore:        synchro.metricsExtraStore,
			WatchListPageSize: synchro.pageSize,
		}
		if clusterpediafeature.FeatureGate.Enabled(features.StreamHandlePaginatedListForResourceSync) {
			config.StreamHandleForPaginatedList = true
		}
		if clusterpediafeature.FeatureGate.Enabled(features.ForcePaginatedListForResourceSync) {
			config.ForcePaginatedList = true
		}
		informer.NewResourceVersionInformer(synchro.cluster, config).Run(informerStopCh)

		// TODO(Iceber): Optimize status updates in case of storage exceptions
		if !synchro.isRunnableForStorage.Load() {
			synchro.setStatus(clusterv1alpha2.ResourceSyncStatusStop, "StorageExpection", "")
		}
	}
}

const LastAppliedConfigurationAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

func (synchro *ResourceSynchro) pruneObject(obj *unstructured.Unstructured) {
	if clusterpediafeature.FeatureGate.Enabled(features.PruneManagedFields) {
		obj.SetManagedFields(nil)
	}

	if clusterpediafeature.FeatureGate.Enabled(features.PruneLastAppliedConfiguration) {
		annotations := obj.GetAnnotations()
		if _, ok := annotations[LastAppliedConfigurationAnnotation]; ok {
			delete(annotations, LastAppliedConfigurationAnnotation)
			if len(annotations) == 0 {
				annotations = nil
			}
			obj.SetAnnotations(annotations)
		}
	}
}

func (synchro *ResourceSynchro) OnAdd(obj interface{}, isInInitialList bool) {
	if !synchro.isRunnableForStorage.Load() {
		return
	}

	// `obj` will not be processed in parallel elsewhere,
	// no deep copy is needed for now.
	//
	// robj := obj.(runtime.Object).DeepCopyObject()

	// Prune object before enqueue.
	//
	// There are many solutions for pruning fields, such as
	// * prunning at the clusterpedia apiserver.
	// * prunning in the storage layer, where neither clustersynchro
	//   nor apiserver are responsible for the pruning process.
	// https://github.com/clusterpedia-io/clusterpedia/issues/4
	synchro.pruneObject(obj.(*unstructured.Unstructured))

	_ = synchro.queue.Add(obj)
}

func (synchro *ResourceSynchro) OnUpdate(_, obj interface{}) {
	if !synchro.isRunnableForStorage.Load() {
		return
	}

	// `obj` will not be processed in parallel elsewhere,
	// no deep copy is needed for now.
	//
	// robj := obj.(runtime.Object).DeepCopyObject()

	// https://github.com/clusterpedia-io/clusterpedia/issues/4
	synchro.pruneObject(obj.(*unstructured.Unstructured))
	_ = synchro.queue.Update(obj)
}

func (synchro *ResourceSynchro) OnDelete(obj interface{}) {
	if !synchro.isRunnableForStorage.Load() {
		return
	}

	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		if obj, ok = d.Obj.(*unstructured.Unstructured); !ok {
			namespace, name, err := cache.SplitMetaNamespaceKey(d.Key)
			if err != nil {
				return
			}
			obj = &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
		}
	}

	if o, ok := obj.(*unstructured.Unstructured); ok {
		synchro.pruneObject(o)
	}

	// full obj is needed in watch feature
	//obj, err := synchro.storage.ConvertDeletedObject(obj)
	//if err != nil {
	//	return
	//}
	_ = synchro.queue.Delete(obj)
}

func (synchro *ResourceSynchro) OnSync(obj interface{}) {}

func (synchro *ResourceSynchro) processResources() {
	for {
		select {
		case <-synchro.closer:
			return
		default:
		}

		event, err := synchro.queue.Pop()
		if err != nil {
			if err == queue.ErrQueueClosed {
				return
			}

			klog.Error(err)
			continue
		}

		synchro.handleResourceEvent(event)
	}
}

func (synchro *ResourceSynchro) handleResourceEvent(event *queue.Event) {
	defer func() { _ = synchro.queue.Done(event) }()

	obj, ok := event.Object.(runtime.Object)
	if !ok {
		if _, ok = event.Object.(cache.DeletedFinalStateUnknown); !ok {
			return
		}
	}
	key, _ := cache.MetaNamespaceKeyFunc(obj)

	var callback func(obj runtime.Object, eventType watch.EventType)
	var handler func(ctx context.Context, obj runtime.Object) (watch.EventType, error)
	if event.Action != queue.Deleted {
		var err error
		if obj, err = synchro.convertToStorageVersion(obj); err != nil {
			klog.ErrorS(err, "Failed to convert resource", "cluster", synchro.cluster,
				"action", event.Action, "resource", synchro.storageResource, "key", key)
			return
		}
		utils.InjectClusterName(obj, synchro.cluster)

		switch event.Action {
		case queue.Added:
			handler = synchro.createOrUpdateResource
		case queue.Updated:
			handler = synchro.updateOrCreateResource
		}
		callback = func(obj runtime.Object, eventType watch.EventType) {
			metaobj, _ := meta.Accessor(obj)
			synchro.rvsLock.Lock()
			synchro.rvs[key] = informer.StorageElement{
				Version:   metaobj.GetResourceVersion(),
				Name:      metaobj.GetName(),
				Namespace: metaobj.GetNamespace(),
				Deleted:   false,
				Published: true,
			}
			synchro.rvsLock.Unlock()
			if middleware.PublisherEnabled {
				err := synchro.storage.ProcessEvent(context.TODO(), eventType, obj, synchro.cluster)
				if err != nil {
					return
				}
			}
		}
	} else {
		handler, callback = synchro.deleteResource, func(_ runtime.Object, eventType watch.EventType) {
			synchro.rvsLock.Lock()
			delete(synchro.rvs, key)
			synchro.rvsLock.Unlock()
			if middleware.PublisherEnabled {
				err := synchro.storage.ProcessEvent(context.TODO(), eventType, obj, synchro.cluster)
				if err != nil {
					return
				}
			}
		}
	}

	// TODO(Iceber): put the event back into the queue to retry?
	for i := 0; ; i++ {
		ctx, cancel := context.WithTimeout(synchro.ctx, 30*time.Second)
		eventType, err := handler(ctx, obj)
		cancel()
		if err == nil {
			callback(obj, eventType)

			if !synchro.isRunnableForStorage.Load() && synchro.queue.Len() == 0 {
				// Start the informer after processing the data in the queue to ensure that storage is up and running for a period of time.
				synchro.setRunnableForStorage()
			}
			return
		}

		if errors.Is(err, context.Canceled) {
			return
		}
		if !storage.IsRecoverableException(err) {
			klog.ErrorS(err, "Failed to storage resource", "cluster", synchro.cluster,
				"action", event.Action, "resource", synchro.storageResource, "key", key)

			if !synchro.isRunnableForStorage.Load() && synchro.queue.Len() == 0 {
				// if the storage returns an error on stopForStorage that cannot be recovered
				// and the len(queue) is empty, start the informer
				synchro.setRunnableForStorage()
			}
			return
		}

		// Store component exceptions, control informer start/stop, and retry sync at regular intervals

		// After five retries, if the data in the queue is greater than 5,
		// keep only 5 items of data in the queue and stop informer to avoid a large accumulation of resources in memory
		var retainInQueue = 5
		if i >= 5 && synchro.queue.Len() > retainInQueue {
			if synchro.isRunnableForStorage.Load() {
				synchro.setStopForStorage()
			}
			synchro.queue.DiscardAndRetain(retainInQueue)

			// If the data in the queue is discarded,
			// the data in the cache will be inconsistent with the data in the `rvs`,
			// delete the cache, and trigger the reinitialization of the cache when the informer is started.
			synchro.rvsLock.Lock()
			synchro.cache = nil
			synchro.rvsLock.Unlock()
		}

		//	klog.ErrorS(err, "will retry sync storage resource", "num", i, "cluster", synchro.cluster,
		//		"action", event.Action, "resource", synchro.storageResource, "key", key)
		time.Sleep(2 * time.Second)
	}
}

func (synchro *ResourceSynchro) setRunnableForStorage() {
	synchro.isRunnableForStorage.Store(true)

	synchro.forStorageLock.Lock()
	defer synchro.forStorageLock.Unlock()

	select {
	case <-synchro.runnableForStorage:
	default:
		close(synchro.runnableForStorage)
	}
	select {
	case <-synchro.stopForStorage:
		synchro.stopForStorage = make(chan struct{})
	default:
	}
}

func (synchro *ResourceSynchro) setStopForStorage() {
	synchro.isRunnableForStorage.Store(false)

	synchro.forStorageLock.Lock()
	defer synchro.forStorageLock.Unlock()

	select {
	case <-synchro.runnableForStorage:
		synchro.runnableForStorage = make(chan struct{})
	default:
	}
	select {
	case <-synchro.stopForStorage:
	default:
		close(synchro.stopForStorage)
	}
}

func (synchro *ResourceSynchro) convertToStorageVersion(obj runtime.Object) (runtime.Object, error) {
	if synchro.syncResource == synchro.storageResource || synchro.convertor == nil {
		return obj, nil
	}

	// convert to hub version
	obj, err := synchro.convertor.ConvertToVersion(obj, synchro.memoryVersion)
	if err != nil {
		return nil, err
	}

	if synchro.memoryVersion == synchro.storageResource.GroupVersion() {
		return obj, nil
	}

	// convert to storage version
	obj, err = synchro.convertor.ConvertToVersion(obj, synchro.storageResource.GroupVersion())
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (synchro *ResourceSynchro) createOrUpdateResource(ctx context.Context, obj runtime.Object) (watch.EventType, error) {
	err := synchro.storage.Create(ctx, synchro.cluster, obj, true)
	if genericstorage.IsExist(err) {
		err = synchro.storage.Update(ctx, synchro.cluster, obj, true)
		return watch.Added, err
	}
	return watch.Added, err
}

func (synchro *ResourceSynchro) updateOrCreateResource(ctx context.Context, obj runtime.Object) (watch.EventType, error) {
	err := synchro.storage.Update(ctx, synchro.cluster, obj, true)
	if genericstorage.IsNotFound(err) {
		err = synchro.storage.Create(ctx, synchro.cluster, obj, true)
		return watch.Modified, err
	}
	return watch.Modified, err
}

func (synchro *ResourceSynchro) deleteResource(ctx context.Context, obj runtime.Object) (watch.EventType, error) {
	err := synchro.storage.Delete(ctx, synchro.cluster, obj, true)
	if err != nil {
		return watch.Deleted, err
	}
	return watch.Deleted, err
}

func (synchro *ResourceSynchro) setStatus(status string, reason, message string) {
	synchro.status.Store(clusterv1alpha2.ClusterResourceSyncCondition{
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now().Rfc3339Copy(),
	})
}

func (synchro *ResourceSynchro) Status() clusterv1alpha2.ClusterResourceSyncCondition {
	return synchro.status.Load().(clusterv1alpha2.ClusterResourceSyncCondition)
}

func (synchro *ResourceSynchro) ErrorHandler(r *informer.Reflector, err error) {
	if err != nil {
		// TODO(iceber): Use `k8s.io/apimachinery/pkg/api/errors` to resolve the error type and update it to `status.Reason`
		synchro.setStatus(clusterv1alpha2.ResourceSyncStatusError, "ResourceWatchFailed", err.Error())
		informer.DefaultWatchErrorHandler(r, err)
		return
	}

	// `reflector` sets a default timeout when watching,
	// then when re-watching the error handler is called again and the `err` is nil.
	// if the current status is Syncing, then the status is not updated to avoid triggering a cluster status update
	if status := synchro.Status(); status.Status != clusterv1alpha2.ResourceSyncStatusSyncing {
		synchro.setStatus(clusterv1alpha2.ResourceSyncStatusSyncing, "", "")
	}
}
