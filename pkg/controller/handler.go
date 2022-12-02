package controller

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/vietanhduong/xcontroller/pkg/apis/foo/v1alpha1"
	"github.com/vietanhduong/xcontroller/pkg/util/log"
)

func (c *Controller) reconcile(bar *v1alpha1.Bar) (err error) {
	if bar.Status.Ready != fmt.Sprintf("%d/%d", bar.Status.ReadyReplicas, bar.Spec.Replicas) {
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			var tmp *v1alpha1.Bar
			if tmp, err = c.barLister.Bars(bar.Namespace).Get(bar.Name); err != nil {
				return err
			}
			tmp.Status.Ready = fmt.Sprintf("%d/%d", bar.Status.ReadyReplicas, bar.Spec.Replicas)
			if tmp, err = c.fooClient.FooV1alpha1().Bars(bar.Namespace).UpdateStatus(c.ctx, tmp, metav1.UpdateOptions{}); err != nil {
				return err
			}
			tmp.DeepCopyInto(bar)
			return nil
		})

		// return current reconcile, wait to the next reconcile
		return err
	}

	var deploy *appsv1.Deployment
	// if the deployment is nil, that means the deployment has been updated, we will wait for the next reconcile
	if deploy, err = c.handleDeployment(bar); err != nil || deploy == nil {
		return err
	}

	return c.onReconcileSuccess(deploy, bar)
}

func (c *Controller) handleDeployment(bar *v1alpha1.Bar) (deploy *appsv1.Deployment, err error) {
	if deploy, err = c.deployLister.Deployments(bar.Namespace).Get(bar.Name); err != nil {
		if errors.IsNotFound(err) {
			_, err = c.kubeClient.AppsV1().Deployments(bar.Namespace).Create(c.ctx, buildDeployment(bar), metav1.CreateOptions{})
			if err != nil {
				log.Errorf("reconcile bar %s/%s: create deployment failed: %v", bar.Namespace, bar.Name, err)
			}
			log.Debugf("reconcile bar %s/%s: create deployment successful", bar.Namespace, bar.Name)
			return nil, err
		}
		log.Errorf("reconcile bar %s/%s: get deployment failed: %v", bar.Namespace, bar.Name, err)
		return nil, err
	}

	var changed bool
	if *deploy.Spec.Replicas != bar.Spec.Replicas {
		deploy.Spec.Replicas = &bar.Spec.Replicas
		changed = true
	}

	handleContainer(deploy, bar, &changed)

	if changed {
		if _, err = c.kubeClient.AppsV1().Deployments(bar.Namespace).Update(c.ctx, deploy, metav1.UpdateOptions{}); err != nil {
			log.Errorf("reconcile bar %s/%s: update deployment failed: %v", bar.Namespace, bar.Name, err)
		}
		return nil, err
	}
	log.Debugf("reconcile bar %s/%s: handle deployment completed with no changed", bar.Namespace, bar.Name)
	return deploy, nil
}

func (c *Controller) onReconcileFailed(bar *v1alpha1.Bar, err error) {
	var msg = err.Error()
	e := retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		var tmp *v1alpha1.Bar
		if tmp, err = c.barLister.Bars(bar.Namespace).Get(bar.Name); err != nil {
			return err
		}

		tmp.Status.Success = false
		tmp.Status.Message = msg
		_, err = c.fooClient.FooV1alpha1().Bars(bar.Namespace).UpdateStatus(c.ctx, tmp, metav1.UpdateOptions{})
		return err
	})

	if e != nil {
		log.Errorf("reconcile bar %/%s (on failed): retry ended with error: %v", e)
	}
}

func (c *Controller) onReconcileSuccess(deploy *appsv1.Deployment, bar *v1alpha1.Bar) (err error) {
	cpStatus := bar.Status.DeepCopy()
	if deploy.Status.ReadyReplicas != bar.Status.ReadyReplicas {
		bar.Status.ReadyReplicas = deploy.Status.ReadyReplicas
		bar.Status.Ready = fmt.Sprintf("%d/%d", bar.Status.ReadyReplicas, bar.Spec.Replicas)
	}

	bar.Status.Success = true
	bar.Status.Message = ""

	if !cmp.Equal(&cpStatus, bar.Status) {
		if _, err = c.fooClient.FooV1alpha1().Bars(bar.Namespace).UpdateStatus(c.ctx, bar, metav1.UpdateOptions{}); err != nil {
			log.Errorf("reconcile bar %s/%s: update bar status failed: %v", bar.Namespace, bar.Name, err)
			return err
		}
		c.recorder.Eventf(bar, "Normal", "Updated", "Bar has been updated!")
	}
	return nil
}

func handleContainer(deploy *appsv1.Deployment, bar *v1alpha1.Bar, changed *bool) {
	i := -1
	for idx, e := range deploy.Spec.Template.Spec.Containers {
		if e.Name == buildContainerName(bar) {
			i = idx
			break
		}
	}

	if i == -1 {
		if deploy.Spec.Template.Spec.Containers == nil {
			deploy.Spec.Template.Spec.Containers = []corev1.Container{}
		}
		deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, buildDeployment(bar).Spec.Template.Spec.Containers[0])
		*changed = true
		return
	}

	container := deploy.Spec.Template.Spec.Containers[i]
	if container.Image != bar.Spec.Image {
		deploy.Spec.Template.Spec.Containers[i].Image = bar.Spec.Image
		*changed = true
	}
}
