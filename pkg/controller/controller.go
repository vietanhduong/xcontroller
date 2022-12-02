package controller

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	app_listers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	foo_scheme "github.com/vietanhduong/xcontroller/pkg/client/clientset/versioned/scheme"
	foo_informers "github.com/vietanhduong/xcontroller/pkg/client/informers/externalversions"
	foo_listers "github.com/vietanhduong/xcontroller/pkg/client/listers/foo/v1alpha1"
	"github.com/vietanhduong/xcontroller/pkg/util/log"

	"github.com/vietanhduong/xcontroller/pkg/apis/foo/v1alpha1"
)

type Controller struct {
	ctx context.Context

	queue workqueue.RateLimitingInterface

	barInformer cache.SharedIndexInformer
	barLister   foo_listers.BarLister

	deployInformer cache.SharedIndexInformer
	deployLister   app_listers.DeploymentLister

	recorder record.EventRecorder
}

func NewController(ctx context.Context,
	fooInformerFactory foo_informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
) *Controller {
	utilruntime.Must(foo_scheme.AddToScheme(scheme.Scheme))
	log.Info("Creating event broadcaster...")

	barInformer := fooInformerFactory.Foo().V1alpha1().Bars()
	deployInformer := kubeInformerFactory.Apps().V1().Deployments()

	controller := &Controller{
		ctx:            ctx,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Bar_queue"),
		barInformer:    barInformer.Informer(),
		barLister:      barInformer.Lister(),
		deployInformer: deployInformer.Informer(),
		deployLister:   deployInformer.Lister(),
		recorder:       recorder,
	}

	controller.barInformer.AddEventHandler(addFooResourceHandlerFunc(controller.queue))
	controller.deployInformer.AddEventHandler(controller.addK8sResourceHandlerFunc())

	return controller
}

func (c *Controller) Run(worker int) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	if ok := cache.WaitForCacheSync(c.ctx.Done(), c.barInformer.HasSynced, c.deployInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < worker; i++ {
		go wait.Until(func() {
			for c.processItemQueue() {
			}
		}, time.Second, c.ctx.Done())
	}

	<-c.ctx.Done()
	return nil
}

func (c *Controller) processItemQueue() bool {
	obj, quit := c.queue.Get()
	if quit {
		return false
	}

	err := func(obj interface{}) error {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("controller/queue: recovered from panic: %+v\n%s", r, debug.Stack())
			}
			c.queue.Done(obj)
		}()

		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.queue.Forget(obj)
			log.Errorf("controller/queue: expected string in workqueue but got %#v", obj)
			return nil
		}

		if err := c.processBar(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("controler/queue: error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		log.Infof("controller/queue: successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		log.Errorf("process queue failed: %v", err)
	}

	return true
}

func (c *Controller) processBar(key string) error {
	var namespace string
	var name string
	var b *v1alpha1.Bar
	var err error

	if namespace, name, err = cache.SplitMetaNamespaceKey(key); err != nil {
		log.Errorf("invalid resource key: '%s'", key)
		return nil
	}

	if b, err = c.barLister.Bars(namespace).Get(name); err != nil {
		// Object already deleted
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return c.reconcile(b)
}

func (c *Controller) addK8sResourceHandlerFunc() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { c.handlerK8sObject(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) {
			o1, ok1 := newObj.(metav1.Object)
			o2, ok2 := oldObj.(metav1.Object)
			if !ok1 || !ok2 {
				log.Infof("Cast object to metav1.Object failed")
				return
			}

			if o1.GetResourceVersion() != o2.GetResourceVersion() {
				c.handlerK8sObject(newObj)
			}

		},
		DeleteFunc: func(obj interface{}) {
			c.handlerK8sObject(obj)
		},
	}
}

func (c *Controller) handlerK8sObject(obj interface{}) {
	var object metav1.Object
	var tombstone cache.DeletedFinalStateUnknown
	var ok bool

	if object, ok = obj.(metav1.Object); !ok {
		if tombstone, ok = obj.(cache.DeletedFinalStateUnknown); !ok {
			log.Errorf("error decoding object, invalid type")
			return
		}

		if object, ok = tombstone.Obj.(metav1.Object); !ok {
			log.Errorf("error decoding object tombstone, invalid type")
			return
		}

		log.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	var owner *metav1.OwnerReference
	if owner = metav1.GetControllerOf(object); owner == nil {
		return
	}

	if owner.Kind == "Bar" {
		var key string
		var b *v1alpha1.Bar
		var err error
		if b, err = c.barLister.Bars(object.GetNamespace()).Get(object.GetName()); err == nil {
			if key, err = cache.MetaNamespaceKeyFunc(b); err == nil {
				c.queue.Add(key)
				return
			}
			log.Errorf("controller: parse metadata key failed: %v", err)
			return
		}

		log.Errorf("controller: get Bar failed: %v", err)
	}
}

func addFooResourceHandlerFunc(queue workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(cur); err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				queue.Add(key)
			}
		},
	}
}
