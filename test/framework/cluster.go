package framework

import (
	"context"
	"log"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/clusterpedia-io/clusterpedia/pkg/generated/clientset/versioned"
	"github.com/clusterpedia-io/clusterpedia/pkg/utils"
)

var (
	crdClient             *versioned.Clientset
	clusterClients        []*ClusterClient
	clusterDynamicClients []*DynamicClusterClient
)

// ClusterClient stands for a cluster Clientset for the given member cluster
type ClusterClient struct {
	KubeClient  *kubernetes.Clientset
	ClusterName string
}

// DynamicClusterClient stands for a dynamic client for the given member cluster
type DynamicClusterClient struct {
	DynamicClientSet dynamic.Interface
	ClusterName      string
}

func InitClusterClient(restconfig *rest.Config, clusters []string) {
	var err error
	crdClient, err = versioned.NewForConfig(restconfig)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	for _, cluster := range clusters {
		pediaCluster, err := crdClient.ClusterV1alpha2().PediaClusters().Get(context.TODO(), cluster, metav1.GetOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		clusterClient := ClusterClient{ClusterName: cluster}
		clusterKubeClient, err := utils.ClusterClientInstance().GetClientSet(pediaCluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		clusterClient.KubeClient = clusterKubeClient
		clusterClients = append(clusterClients, &clusterClient)

		dynamicClient := DynamicClusterClient{ClusterName: cluster}
		clusterDynamicClient, err := utils.ClusterClientInstance().GetDynamicClientSet(pediaCluster, 100)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		dynamicClient.DynamicClientSet = clusterDynamicClient
		clusterDynamicClients = append(clusterDynamicClients, &dynamicClient)
	}
}

// GetClusterClient get cluster client
func GetClusterClient(clusterName string) kubernetes.Interface {
	for _, clusterClient := range clusterClients {
		if clusterClient.ClusterName == clusterName {
			return clusterClient.KubeClient
		}
	}

	return nil
}

// GetClusterDynamicClient get cluster dynamicClient
func GetClusterDynamicClient(clusterName string) dynamic.Interface {
	for _, clusterClient := range clusterDynamicClients {
		if clusterClient.ClusterName == clusterName {
			return clusterClient.DynamicClientSet
		}
	}

	return nil
}

// LoadRESTClientConfig creates a rest.Config using the passed kubeconfig. If context is empty, current context in kubeconfig will be used.
func LoadRESTClientConfig(kubeconfig string, context string) (*rest.Config, error) {
	loader := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	loadedConfig, err := loader.Load()
	if err != nil {
		return nil, err
	}

	if context == "" {
		context = loadedConfig.CurrentContext
	}
	klog.Infof("Use context %v", context)

	return clientcmd.NewNonInteractiveClientConfig(
		*loadedConfig,
		context,
		&clientcmd.ConfigOverrides{},
		loader,
	).ClientConfig()
}

func GetMemberConfig(cluster string) (*rest.Config, error) {
	pediaCluster, err := crdClient.ClusterV1alpha2().PediaClusters().Get(context.TODO(), cluster, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("can not find pediacluster : %v  by error : %v", cluster, err)
		return nil, err
	}

	memberConfig, err := utils.BuildClusterConfig(pediaCluster)
	if err != nil {
		log.Fatalf("rcan not get member  %v kubeconfig : %v", cluster, err)
		return nil, err
	}
	return memberConfig, nil
}
