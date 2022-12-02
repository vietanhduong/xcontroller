package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vietanhduong/xcontroller/pkg/apis/foo/v1alpha1"
)

func buildDeployment(bar *v1alpha1.Bar) *appsv1.Deployment {
	labels := buildLabels(bar)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            bar.Name,
			Namespace:       bar.Namespace,
			Labels:          labels,
			Annotations:     bar.Spec.Annotations,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(bar, v1alpha1.SchemeGroupVersion.WithKind("Bar"))},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &bar.Spec.Replicas,
			Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: bar.Spec.Annotations,
					Labels:      labels,
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  buildContainerName(bar),
					Image: bar.Spec.Image,
				}}},
			},
		},
	}
}

func buildLabels(bar *v1alpha1.Bar) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       bar.Name,
		"app.kubernetes.io/managed-by": "xcontroller",
		"app.kubernetes.io/instance":   bar.Name,
	}
}

func buildContainerName(bar *v1alpha1.Bar) string {
	if bar.Spec.ContainerName != "" {
		return bar.Spec.ContainerName
	}
	return bar.Name
}
