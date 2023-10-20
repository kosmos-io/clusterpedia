package framework

import (
	"context"
	"errors"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	"github.com/clusterpedia-io/clusterpedia/test/helper"
)

// CreateDeployment create Deployment.
func CreateDeploymentOnClusterPedia(client kubernetes.Interface, deployment *appsv1.Deployment) {
	ginkgo.By(fmt.Sprintf("Creating Deployment(%s/%s)", deployment.Namespace, deployment.Name), func() {
		_, err := client.AppsV1().Deployments(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("create deployment occur error ：", err)
			gomega.Expect(apierrors.IsAlreadyExists(err)).Should(gomega.Equal(true))
		}
	})
}

func RemoveDeploymentOnClusterPedia(client kubernetes.Interface, namespace, name string) {
	ginkgo.By(fmt.Sprintf("Removing Deployment(%s/%s)", namespace, name), func() {
		err := client.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

func RemoveDeploymentsOnClusterPediaFitException(client kubernetes.Interface, namespace string, opts metav1.DeleteOptions, listOpt metav1.ListOptions) {
	ginkgo.By(fmt.Sprintf("Removing Deployments(%v/%v)", namespace, opts), func() {
		err := client.AppsV1().Deployments(namespace).DeleteCollection(context.TODO(), opts, listOpt)

		expectedError := errors.New("[Delete-Collection]: exist multiple-cluster have namespace default ")
		gomega.Expect(err.Error()).To(gomega.Equal(expectedError.Error()))
	})
}

func RemoveDeploymentsOnClusterPedia(client kubernetes.Interface, namespace string, opts metav1.DeleteOptions, listOpt metav1.ListOptions) {
	ginkgo.By(fmt.Sprintf("Removing Deployments(%v/%v)", namespace, opts), func() {
		err := client.AppsV1().Deployments(namespace).DeleteCollection(context.TODO(), opts, listOpt)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

func CreateDeploymentOnCluster(cluster string, deployment *appsv1.Deployment) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	_, err := clusterClient.AppsV1().Deployments(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func CreateNamespaceOnCluster(cluster string, namespace *corev1.Namespace) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	_, err := clusterClient.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func RemoveDeploymentOnCluster(cluster, namespace, name string) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	ginkgo.By(fmt.Sprintf("Removing Deployment(%s/%s)", namespace, name), func() {
		err := clusterClient.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

func RemoveNamespaceOnCluster(cluster, name string) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	ginkgo.By(fmt.Sprintf("Removing Namespace(%s)", name), func() {
		err := clusterClient.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

func WaitDeploymentDisappearOnCluster(cluster, namespace, name string) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	gomega.Eventually(func() bool {
		_, err := clusterClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return false
		}
		if apierrors.IsNotFound(err) {
			return true
		}

		klog.Errorf("Failed to get deployment(%s/%s) on cluster(%s), err: %v", namespace, name, cluster, err)
		return false
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}

func WaitDeploymentDisappearOnClusterPedia(client kubernetes.Interface, namespace, name string) {
	gomega.Eventually(func() bool {
		_, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return false
		}
		if apierrors.IsNotFound(err) {
			return true
		}

		klog.Errorf("Failed to get deployment(%s/%s) on clusterpedia, err: %v", namespace, name, err)
		return false
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}

func WaitDeploymentPresentOnClusterFitWith(cluster, namespace, name string, fit func(deployment *appsv1.Deployment) bool) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())

	klog.Infof("Waiting for deployment(%s/%s) on cluster(%s)", namespace, name, cluster)
	gomega.Eventually(func() bool {
		dep, err := clusterClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return fit(dep)
	}, PollTimeout, PollInterval).Should(gomega.Equal(true))
}

func WaitDeploymentPresentOnClusterPediaFitWith(pediaClient *kubernetes.Clientset, namespace, name string, fit func(deployment *appsv1.Deployment) bool) {
	ginkgo.By(fmt.Sprintf("Waiting for deployment(%s/%s) on clusterpedia", namespace, name), func() {
		gomega.Eventually(func() bool {
			dep, err := pediaClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return fit(dep)
		}, PollTimeout, PollInterval).Should(gomega.Equal(true))
	})
}

func GetDeploymentOnClusterPedia(pediaClient *kubernetes.Clientset, namespace, name string) *appsv1.Deployment {
	dep, err := pediaClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	return dep
}

func GetDeploymentOnCluster(cluster, namespace, name string) *appsv1.Deployment {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	dep, err := clusterClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	return dep
}

func UpdateDeploymentOnClusterpedia(pediaClient *kubernetes.Clientset, namespace string, deploy *appsv1.Deployment) *appsv1.Deployment {
	updated := &appsv1.Deployment{}
	err := retry.RetryOnConflict(helper.DefinedRetry, func() error {
		latestDeploy, err := pediaClient.AppsV1().Deployments(namespace).Get(context.TODO(), deploy.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		latestDeploy.Spec = deploy.Spec
		deployed, err := pediaClient.AppsV1().Deployments(namespace).Update(context.TODO(), latestDeploy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		updated = deployed.DeepCopy()
		return nil
	})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	return updated
}

func UpdateDeploymentOnCluster(cluster, namespace string, deploy *appsv1.Deployment) {
	clusterClient := GetClusterClient(cluster)
	gomega.Expect(clusterClient).ShouldNot(gomega.BeNil())
	err := retry.RetryOnConflict(helper.DefinedRetry, func() error {
		latestDeploy, err := clusterClient.AppsV1().Deployments(namespace).Get(context.TODO(), deploy.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		latestDeploy.Spec = deploy.Spec
		_, err = clusterClient.AppsV1().Deployments(namespace).Update(context.TODO(), latestDeploy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}
