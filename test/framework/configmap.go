package framework

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func CreateConfigMapOnClusterPedia(client kubernetes.Interface, configMap *v1.ConfigMap) {
	ginkgo.By(fmt.Sprintf("Creating ConfigMap(%s/%s)", configMap.Namespace, configMap.Name), func() {
		_, err := client.CoreV1().ConfigMaps(configMap.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			gomega.Expect(apierrors.IsAlreadyExists(err)).Should(gomega.Equal(true))
		}
	})
}

func WaitConfigMapPresentOnClusterPediaFitWith(pediaClient *kubernetes.Clientset, namespace, name string, fit func(configMap *v1.ConfigMap) bool) {
	ginkgo.By(fmt.Sprintf("Waiting for configMap(%s/%s) on clusterpedia", namespace, name), func() {
		gomega.Eventually(func() bool {
			configMap, err := pediaClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return fit(configMap)
		}, PollTimeout, PollInterval).Should(gomega.Equal(true))
	})
}

func RemoveConfigMapOnClusterPedia(client kubernetes.Interface, namespace, name string) {
	ginkgo.By(fmt.Sprintf("Removing ConfigMap(%s/%s)", namespace, name), func() {
		err := client.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

func WaitConfigMapDisappearOnCluster(cluster, namespace, name string) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	gomega.Eventually(func() bool {
		_, err := clusterClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return false
		}
		if apierrors.IsNotFound(err) {
			return true
		}

		klog.Errorf("Failed to get configMap(%s/%s) on cluster(%s), err: %v", namespace, name, cluster, err)
		return false
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}

func WaitConfigMapDisappearOnClusterPedia(pediaClient *kubernetes.Clientset, namespace, name string) {
	gomega.Eventually(func() bool {
		_, err := pediaClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return false
		}
		if apierrors.IsNotFound(err) {
			return true
		}

		klog.Errorf("Failed to get configMap(%s/%s) on clusterpedia, err: %v", namespace, name, err)
		return false
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}

func WaitConfigMapPresentOnClusterFitWith(cluster, namespace, name string, fit func(configMap *v1.ConfigMap) bool) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())

	klog.Infof("Waiting for configMap(%s/%s) on cluster(%s)", namespace, name, cluster)
	gomega.Eventually(func() bool {
		configMap, err := clusterClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return fit(configMap)
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}
