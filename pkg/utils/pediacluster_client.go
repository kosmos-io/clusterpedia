package utils

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	clusterv1alpha2 "github.com/clusterpedia-io/api/cluster/v1alpha2"
)

var (
	clientMap          map[string]*kubeclientset.Clientset
	dynamicClientMap   map[string]dynamic.Interface
	clientSetconfigMap map[string]*rest.Config
	instance           *ClientManager
	once               sync.Once
)

type ClientManager struct {
	clientMap          map[string]*kubeclientset.Clientset
	dynamicClientMap   map[string]dynamic.Interface
	clientSetconfigMap map[string]*rest.Config
	lock               sync.RWMutex
}

func init() {
	clientMap = make(map[string]*kubeclientset.Clientset)
	dynamicClientMap = make(map[string]dynamic.Interface)
	clientSetconfigMap = make(map[string]*rest.Config)
}

func ClusterClientInstance() *ClientManager {
	once.Do(func() {
		instance = &ClientManager{
			clientMap:          clientMap,
			dynamicClientMap:   dynamicClientMap,
			clientSetconfigMap: clientSetconfigMap,
		}
	})
	return instance
}

func (m *ClientManager) GetClientSet(cluster *clusterv1alpha2.PediaCluster) (*kubeclientset.Clientset, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var err error
	var config *rest.Config
	clientset, exist := m.clientMap[cluster.Name]
	if !exist {
		clientset, config, err = NewClusterClientSet(cluster)
		if err != nil {
			return nil, err
		}
		m.clientMap[cluster.Name] = clientset
		m.clientSetconfigMap[cluster.Name] = config
	}

	return clientset, nil
}

func (m *ClientManager) GetClientSetConfig(cluster *clusterv1alpha2.PediaCluster) (*rest.Config, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	config, exist := m.clientSetconfigMap[cluster.Name]
	if !exist {
		return nil, fmt.Errorf("cluster %v clientset config doesn't exit", cluster.Name)
	}
	return config, nil
}

func (m *ClientManager) GetDynamicClientSet(cluster *clusterv1alpha2.PediaCluster, clusterQPS float32) (dynamic.Interface, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var err error
	clientset, exist := m.dynamicClientMap[cluster.Name]
	if !exist {
		clientset, err = NewClusterDynamicClientSet(cluster, clusterQPS)
		if err != nil {
			return nil, err
		}
		m.dynamicClientMap[cluster.Name] = clientset
	}

	return clientset, nil
}

// NewClusterClientSet returns a ClusterClient for the given member cluster.
func NewClusterClientSet(cluster *clusterv1alpha2.PediaCluster) (*kubeclientset.Clientset, *rest.Config, error) {
	clusterConfig, err := BuildClusterConfig(cluster)
	if err != nil {
		return nil, nil, err
	}

	if clusterConfig != nil {
		return kubeclientset.NewForConfigOrDie(clusterConfig), clusterConfig, nil
	}

	return nil, nil, errors.New("empty cluster config error")
}

// NewClusterDynamicClientSet returns a dynamic client for the given member cluster.
func NewClusterDynamicClientSet(cluster *clusterv1alpha2.PediaCluster, clusterQPS float32) (dynamic.Interface, error) {
	clusterConfig, err := BuildClusterConfig(cluster)
	if err != nil {
		return nil, err
	}

	if clusterConfig != nil {
		clusterConfig.QPS = clusterQPS
		return dynamic.NewForConfigOrDie(clusterConfig), nil
	}
	return nil, errors.New("empty cluster config error")
}

func BuildClusterConfig(cluster *clusterv1alpha2.PediaCluster) (*rest.Config, error) {
	if len(cluster.Spec.Kubeconfig) != 0 {
		clientconfig, err := clientcmd.NewClientConfigFromBytes(cluster.Spec.Kubeconfig)
		if err != nil {
			return nil, err
		}
		return clientconfig.ClientConfig()
	}

	if cluster.Spec.APIServer == "" {
		return nil, errors.New("Cluster APIServer Endpoint is required")
	}

	if len(cluster.Spec.TokenData) == 0 &&
		(len(cluster.Spec.CertData) == 0 || len(cluster.Spec.KeyData) == 0) {
		return nil, errors.New("Cluster APIServer's Token or Cert is required")
	}

	config := &rest.Config{
		Host: cluster.Spec.APIServer,
	}

	if len(cluster.Spec.CAData) != 0 {
		config.TLSClientConfig.CAData = cluster.Spec.CAData
	} else {
		config.TLSClientConfig.Insecure = true
	}

	if len(cluster.Spec.CertData) != 0 && len(cluster.Spec.KeyData) != 0 {
		config.TLSClientConfig.CertData = cluster.Spec.CertData
		config.TLSClientConfig.KeyData = cluster.Spec.KeyData
	}

	if len(cluster.Spec.TokenData) != 0 {
		config.BearerToken = string(cluster.Spec.TokenData)
	}
	return config, nil
}
