package rabbitmq

import (
	"context"
	watchComponents "github.com/clusterpedia-io/clusterpedia/pkg/watcher/components"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

type RabbitmqPublisher struct {
}

func (r *RabbitmqPublisher) InitPublisher(ctx context.Context) error {
	return nil
}

func (r *RabbitmqPublisher) PublishTopic(gvr schema.GroupVersionResource, codec runtime.Codec) error {
	return nil
}

func (r *RabbitmqPublisher) EventSending(gvr schema.GroupVersionResource, startChan func(schema.GroupVersionResource) chan *watchComponents.EventWithCluster,
	publishEvent func(context.Context, *watchComponents.EventWithCluster), GenCrv2Event func(event *watch.Event)) error {
	return nil
}

func (r *RabbitmqPublisher) StopPublishing(gvr schema.GroupVersionResource) error {
	return nil
}

func (r *RabbitmqPublisher) StopPublisher() {

}