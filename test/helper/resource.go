package helper

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	internal "github.com/clusterpedia-io/api/clusterpedia"
)

const LabelName = "appName"

func NewDeployment(cluster, namespace, name, rand string) *appsv1.Deployment {
	deployLabels := map[string]string{"app": "nginx", LabelName: "test-instance-id-" + rand}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    deployLabels,
			Annotations: map[string]string{
				internal.ShadowAnnotationClusterName: cluster,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: deployLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: deployLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "registry.paas/cmss/nginx:1.21.4",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceCPU: resource.MustParse("100m"),
							},
						},
					}},
				},
			},
		},
	}
}

func NewNameSpaceWithLabel(name, labelName, labelStr string) *corev1.Namespace {
	resourceLabels := map[string]string{labelName: labelStr}
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: resourceLabels,
		},
	}
}

func NewNameSpaceWithFailed(name string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// NewConfigMap creates a new static configmap
func NewConfigMap(namespace, name, rand string) *corev1.ConfigMap {
	configMapLabels := map[string]string{LabelName: "test-instance-id-" + rand}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    configMapLabels,
		},
		Data: map[string]string{
			"username": "chinaMobile",
			"paaword":  "123",
		},
	}
}

// NewNameSpace creates a new static namespace
func NewNameSpace(name, rand string) *corev1.Namespace {
	nameSpaceLabels := map[string]string{LabelName: "test-instance-id-" + rand}
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"test": "successfully",
			},
			Labels: nameSpaceLabels,
		},
	}
}

func NewService(cluster, namespace, name, rand string) *corev1.Service {
	svcLabels := map[string]string{"app": "nginx", LabelName: "test-instance-id-" + rand}

	svcPort := corev1.ServicePort{
		Name:       "test-port",
		Protocol:   "TCP",
		Port:       32700,
		TargetPort: intstr.FromInt(32700),
		NodePort:   32700,
	}

	ports := []corev1.ServicePort{
		svcPort,
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "v1",
			APIVersion: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    svcLabels,
			Annotations: map[string]string{
				internal.ShadowAnnotationClusterName: cluster,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeNodePort,
			Ports: ports,
		},
	}
}
