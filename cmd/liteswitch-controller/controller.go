package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/WesleyEvanMiller/Liteswitch/handler"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appsListers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/scheme"
)

const controllerAgentName = "liteswitch-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Deployment is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Deployment fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Deployment"
	// MessageResourceSynced is the message used for an Event fired when a Deployment
	// is synced successfully
	MessageResourceSynced = "Deployment synced successfully"
)

var (
	resourceQuotaValues corev1.ResourceList
)

// Controller is the controller implementation for liteswitch-controller
type Controller struct {
	config *config
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	handlers []handler.Handler

	deploymentsLister appsListers.DeploymentLister
	deploymentsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// NewController returns a new deployment controller
func NewController(kubeclientset kubernetes.Interface, deploymentInformer appsinformers.DeploymentInformer, configData io.Reader) (*Controller, error) {
	klog.V(4).Infof("Creating %s", controllerAgentName)

	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	config, err := initDefaults(configData)
	if err != nil {
		return &Controller{}, err
	}

	h := handler.NewHandlers(kubeclientset, config.handlerConfig)

	controller := &Controller{
		config:            config,
		kubeclientset:     kubeclientset,
		handlers:          h,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Deployments"),
		recorder:          recorder,
	}

	klog.Info("Setting up event handlers")
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			controller.enqueueDeployment(deployment)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			newDeployment := new.(*appsv1.Deployment)
			oldDeployment := old.(*appsv1.Deployment)

			if oldDeployment == newDeployment {
				return
			}

			controller.enqueueDeployment(newDeployment)
		},
		DeleteFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			controller.enqueueDeployment(deployment)
		},
	})

	return controller, nil
}

// Run will set up the event handlers for types we are interested in
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Infof("Starting %s", controllerAgentName)

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch workers to process resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the deployment/name string of the

		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Deployment resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	deploymentNamespace, deploymentName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	//Get namespace to see if the deployment is in a NS disabled where this controller is disabled.
	namespace, err := c.kubeclientset.CoreV1().Namespaces().Get(context.TODO(), deploymentNamespace, v1.GetOptions{})
	if err != nil {
		return err
	}

	//If namespace is defined as omitted in the config or the Namespace was given a disabled label return err.
	_, ok := c.config.nsToOmit[namespace.Name]
	if namespace.Labels["liteswtich-status"] == "disabled" || ok {
		if ok {
			klog.Infof("The deployment in namespace %s is omitted from sync", deploymentNamespace)
			return nil
		} else if namespace.Labels["liteswtich-status"] == "disabled" {
			klog.Infof("Liteswtich is disabled in the %s namespace", deploymentNamespace)
			return nil
		} else {
			return err
		}
	}

	//Get config labels for the node pool you wish to be controlled. These are custom to your design.
	nodePoolLabel := c.config.nodePoolLabel

	// Get the deployment from the informer.
	deployment, err := c.deploymentsLister.Deployments(deploymentNamespace).Get(deploymentName)
	if err != nil {
		// The deployment resource may have been deleted or doesn't exist, in which case we handle cleaning if needed.
		if errors.IsNotFound(err) {

			// Create a deployment requirement for grabbing deployments by key.
			deploymentLabelReq, err := labels.NewRequirement(nodePoolLabel.Key, selection.Equals, []string{nodePoolLabel.Value})
			if err != nil {
				return err
			}

			deploymentSelector := labels.NewSelector()
			deploymentSelector = deploymentSelector.Add(*deploymentLabelReq)

			// Get a list of deployments that share the selector keys given by the config file
			deploymentList, err := c.deploymentsLister.List(deploymentSelector)
			if err != nil {
				return err
			}

			// If no other deployments have the NP controller key, run the clean method to stop the node pool
			if len(deploymentList) == 0 {
				for _, h := range c.handlers {
					err := h.Clean()
					if err != nil {
						klog.Error(err)
					}
				}
				return err
			} else {
				utilruntime.HandleError(fmt.Errorf("Deployment '%s' (Namespace '%s') in work queue no longer exists. However, other deployments are present with the label '%s': '%s' as defined in the config. The node pool will not be stopped.", deploymentName, deploymentNamespace, nodePoolLabel.Key, nodePoolLabel.Value))
				return nil
			}
		}

		return err
	}

	//Check if our deployment has that label
	if deployment.Labels[nodePoolLabel.Key] == nodePoolLabel.Value {

		//Else namespace is not omitted and we sync the deployment
		for _, h := range c.handlers {
			err = h.Sync(*deployment)
			if err != nil {
				klog.Error(err)
			}
		}

		return err
	}

	return err
}

// enqueueDeployment takes a deployment resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Deployment.
// enqueueDeployment adds an object to the controller work queue
// obj could be an *v1.deployment, or a DeletionFinalStateUnknown item.
func (c *Controller) enqueueDeployment(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}
