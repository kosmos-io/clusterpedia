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

func CreateNameSpaceOnClusterPedia(client kubernetes.Interface, nameSpace *v1.Namespace) {
	ginkgo.By(fmt.Sprintf("Creating NameSpace(%s)", nameSpace.Name), func() {
		_, err := client.CoreV1().Namespaces().Create(context.TODO(), nameSpace, metav1.CreateOptions{})
		if err != nil {
			gomega.Expect(apierrors.IsAlreadyExists(err)).Should(gomega.Equal(true))
		}
	})
}

func WaitNameSpacePresentOnClusterPediaFitWith(pediaClient *kubernetes.Clientset, name string, fit func(nameSpace *v1.Namespace) bool) {
	ginkgo.By(fmt.Sprintf("Waiting for nameSpace(%s) on clusterpedia", name), func() {
		gomega.Eventually(func() bool {
			nameSpace, err := pediaClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return fit(nameSpace)
		}, PollTimeout, PollInterval).Should(gomega.Equal(true))
	})
}

func WaitNameSpacePresentOnClusterFitWith(cluster, name string, fit func(nameSpace *v1.Namespace) bool) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())

	klog.Infof("Waiting for nameSpace(%s) on cluster(%s)", name, cluster)
	gomega.Eventually(func() bool {
		nameSpace, err := clusterClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return fit(nameSpace)
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}

func RemoveNameSpaceOnClusterPedia(client kubernetes.Interface, name string) {
	ginkgo.By(fmt.Sprintf("Removing nameSpace(%s) on clusterpedia", name), func() {
		err := client.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

func WaitNameSpaceDisappearOnClusterPedia(pediaClient *kubernetes.Clientset, name string) {
	gomega.Eventually(func() bool {
		_, err := pediaClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return false
		}
		if apierrors.IsNotFound(err) {
			return true
		}

		klog.Errorf("Failed to get nameSpace(%s) on clusterpedia, err: %v", name, err)
		return false
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}

func WaitNameSpaceDisappearOnCluster(cluster, name string) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	gomega.Eventually(func() bool {
		_, err := clusterClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return false
		}
		if apierrors.IsNotFound(err) {
			return true
		}

		klog.Errorf("Failed to get nameSpace(%s) on cluster(%s), err: %v", name, cluster, err)
		return false
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}
