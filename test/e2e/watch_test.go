package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/clusterpedia-io/clusterpedia/test/framework"
	"github.com/clusterpedia-io/clusterpedia/test/helper"
)

var (
	watchRandomStr   = helper.RandomString()
	watchCluster     = "business-1-14-k8s"
	watchDeployName  = "watch-deploy-" + watchRandomStr
	watchNs          = "default"
	watchRestConfig  *rest.Config
	watchPediaClient *kubernetes.Clientset
	result           watch.Interface
	watchTimeout     = time.Second * 60 // 等待deployment的所有pod ready的时间
)

var _ = ginkgo.Describe("Watch test for deployment", func() {
	ginkgo.BeforeEach(func() {
		var err error
		// init
		watchRestConfig, err = framework.LoadRESTClientConfig(kubeconfig, "")
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		framework.InitClusterClient(watchRestConfig, []string{watchCluster})

		watchRestConfig.Host = fmt.Sprintf("%s/apis/clusterpedia.io/v1beta1/resources/clusters/%s", apiserverAddress, watchCluster)
		watchPediaClient, err = kubernetes.NewForConfig(watchRestConfig)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
	})

	// 测试从子集群创建资源生成的事件
	ginkgo.Context("Test watch deployments", func() {
		var err error
		var watchResult bool
		ginkgo.BeforeEach(func() {
			result, err = watchPediaClient.AppsV1().Deployments(watchNs).Watch(context.TODO(), metav1.ListOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			ginkgo.DeferCleanup(func() {
				// 可能在spec中已经删除了deployment
				deployNames := [4]string{"-add", "-detail", "-update", "-delete"}
				for _, name := range deployNames {
					tagName := fmt.Sprintf("%s%s", watchDeployName, name)
					clusterClient := framework.GetClusterClient(watchCluster)
					_, err := clusterClient.AppsV1().Deployments(watchNs).Get(context.TODO(), tagName, metav1.GetOptions{})
					if err == nil {
						framework.RemoveDeploymentOnCluster(watchCluster, watchNs, tagName)
					}
				}
			})
		})
		ginkgo.It("test event generate from member cluster", func() {
			watchDeployNameAdd := fmt.Sprintf("%s-add", watchDeployName)
			addCallBack := func(event watch.Event) bool {
				deploy := event.Object.(*v1.Deployment)
				if event.Type == watch.Added && deploy.Name == watchDeployNameAdd {
					return true
				}
				return false
			}

			ginkgo.By("test add event from member cluster", func() {
				deploy := helper.NewDeployment(watchCluster, watchNs, watchDeployNameAdd, rand.String(6))
				framework.CreateDeploymentOnCluster(watchCluster, deploy)
				watchResult, err = GetEvent(result, watchTimeout, addCallBack)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(watchResult).Should(gomega.Equal(true))
			})

			watchDeployNameDetail := watchDeployName + "-detail"
			detailCallBack := func(event watch.Event) bool {
				deploy := event.Object.(*v1.Deployment)
				if event.Type == watch.Modified && deploy.Name == watchDeployNameDetail && deploy.Status.ReadyReplicas == 1 {
					return true
				}
				return false
			}

			ginkgo.By("test event detail from member cluster", func() {
				deploy := helper.NewDeployment(watchCluster, watchNs, watchDeployNameDetail, rand.String(6))
				framework.CreateDeploymentOnCluster(watchCluster, deploy)
				watchResult, err = GetEvent(result, watchTimeout, detailCallBack)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(watchResult).Should(gomega.Equal(true))
			})

			watchDeployNameUpdate := fmt.Sprintf("%s-update", watchDeployName)
			updateCallBack := func(event watch.Event) bool {
				deploy := event.Object.(*v1.Deployment)
				if event.Type == watch.Modified && deploy.Name == watchDeployNameUpdate && deploy.Status.ReadyReplicas == 2 {
					return true
				}
				return false
			}

			ginkgo.By("test update event from member cluster", func() {
				deploy := helper.NewDeployment(watchCluster, watchNs, watchDeployNameUpdate, rand.String(6))
				framework.CreateDeploymentOnCluster(watchCluster, deploy)
				framework.WaitDeploymentPresentOnClusterFitWith(watchCluster, watchNs, watchDeployNameUpdate, func(deployment *v1.Deployment) bool {
					return true
				})
				replicas := int32(2)
				deploy.Spec.Replicas = &replicas
				framework.UpdateDeploymentOnCluster(watchCluster, watchNs, deploy)
				watchResult, err = GetEvent(result, watchTimeout, updateCallBack)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(watchResult).Should(gomega.Equal(true))
			})

			watchDeployNameDelete := fmt.Sprintf("%s-delete", watchDeployName)
			deleteCallBack := func(event watch.Event) bool {
				deploy := event.Object.(*v1.Deployment)
				return event.Type == watch.Deleted && deploy.Name == watchDeployNameDelete
			}

			ginkgo.By("test deleted event from member cluster", func() {
				deploy := helper.NewDeployment(watchCluster, watchNs, watchDeployNameDelete, rand.String(6))
				framework.CreateDeploymentOnCluster(watchCluster, deploy)
				framework.WaitDeploymentPresentOnClusterFitWith(watchCluster, watchNs, watchDeployNameDelete, func(deployment *v1.Deployment) bool {
					return true
				})
				framework.RemoveDeploymentOnCluster(watchCluster, watchNs, watchDeployNameDelete)
				watchResult, err = GetEvent(result, watchTimeout, deleteCallBack)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(watchResult).Should(gomega.Equal(true))
			})

		})
	})

	ginkgo.Context("Test watch namespace with labelSelector", func() {
		var err error
		var watchResult bool
		var watchNsStr = watchNs + "-" + watchRandomStr
		labelStr := "e2e-test-namespace-watch-" + rand.String(6)
		labelName := "app"
		ginkgo.BeforeEach(func() {
			result, err = watchPediaClient.CoreV1().Namespaces().Watch(context.TODO(), metav1.ListOptions{
				LabelSelector: labelName + "=" + labelStr,
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			ginkgo.DeferCleanup(func() {
				// 可能在spec中已经删除了deployment
				clusterClient := framework.GetClusterClient(watchCluster)
				_, err := clusterClient.CoreV1().Namespaces().Get(context.TODO(), watchNsStr, metav1.GetOptions{})
				if err == nil {
					framework.RemoveNamespaceOnCluster(watchCluster, watchNsStr)
				}
			})
		})

		ginkgo.It("test event of namespace which generate from member cluster", func() {
			namespaceCallBack := func(event watch.Event) bool {
				namespace := event.Object.(*corev1.Namespace)
				if event.Type == watch.Added && namespace.Name == watchNsStr {
					return true
				}
				return false
			}

			namespace := helper.NewNameSpaceWithLabel(watchNsStr, labelName, labelStr)
			framework.CreateNamespaceOnCluster(watchCluster, namespace)
			watchResult, err = GetEvent(result, watchTimeout, namespaceCallBack)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(watchResult).Should(gomega.Equal(true))
		})

	})

	ginkgo.Context("Test watch namespace with FieldSelector", func() {
		var err error
		var watchResult bool
		var watchNsStr = watchNs + "-" + watchRandomStr + "-1"
		ginkgo.BeforeEach(func() {
			result, err = watchPediaClient.CoreV1().Namespaces().Watch(context.TODO(), metav1.ListOptions{
				FieldSelector: "metadata.name=" + watchNsStr,
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			ginkgo.DeferCleanup(func() {
				// 可能在spec中已经删除了deployment
				clusterClient := framework.GetClusterClient(watchCluster)
				_, err := clusterClient.CoreV1().Namespaces().Get(context.TODO(), watchNsStr, metav1.GetOptions{})
				if err == nil {
					framework.RemoveNamespaceOnCluster(watchCluster, watchNsStr)
				}
			})
		})

		ginkgo.It("test event of namespace which generate from member cluster", func() {
			namespaceCallBack := func(event watch.Event) bool {
				namespace := event.Object.(*corev1.Namespace)
				if event.Type == watch.Added && namespace.Name == watchNsStr {
					return true
				}
				return false
			}

			namespace := helper.NewNameSpaceWithFailed(watchNsStr)
			framework.CreateNamespaceOnCluster(watchCluster, namespace)
			watchResult, err = GetEvent(result, watchTimeout, namespaceCallBack)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(watchResult).Should(gomega.Equal(true))
		})

	})

})

func GetEvent(w watch.Interface, duration time.Duration, test func(watch.Event) bool) (bool, error) {
loop:
	for {
		select {
		case event, ok := <-w.ResultChan():
			if !ok {
				break loop
			}
			if event.Type == watch.Error {
				// todo 遇到error是否需要退出？
				return false, fmt.Errorf("event type is Error,event is %v", event.Object)
			}
			if flag := test(event); flag == true {
				return true, nil
			}
		case <-time.After(duration):
			return false, fmt.Errorf("wait resource status of running time out")
		}
	}
	return false, fmt.Errorf("get event from result chan error")
}
