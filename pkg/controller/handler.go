package controller

import (
	"k8s.io/klog/v2"

	"github.com/vietanhduong/xcontroller/pkg/apis/foo/v1alpha1"
)

func (c *Controller) reconcile(fb *v1alpha1.Bar) error {
	klog.Infof("Bar: %v", fb)
	return nil
}
