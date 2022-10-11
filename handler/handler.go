package handler

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// Check interface implementation at compile time
var _ Handler = (*NodePoolStateHandler)(nil)

type CloudCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
}

type NodePool struct {
	Name                 string `json:"name"`
	Cluster              string `json:"cluster"`
	ClusterResourceGroup string `json:"clusterResourceGroup"`
}

// Handler is interface that every handler must
// Sync is designed to sync you desired state depending on the NS config
// Sync should always have the ability to disable
// Clean should always run
type Handler interface {
	Sync(deployment appsv1.Deployment) error
	Clean() error
	isEnabled() bool
}

// Config ...
type Config struct {
	NodePoolStateHandlerEnabled bool
	CloudType                   string
	CloudGroupID                string
	CloudCredentials            CloudCredentials
	NodePool                    NodePool
}

func NewHandlers(kubeclient kubernetes.Interface, config Config) []Handler {
	var handlers []Handler

	handlers = append(handlers,
		NodePoolStateHandler{
			kubeclient: kubeclient,
			config:     config,
		},
	)

	return handlers
}

type NodePoolStateHandler struct {
	kubeclient kubernetes.Interface
	config     Config
}

func (h NodePoolStateHandler) isEnabled() bool {
	return h.config.NodePoolStateHandlerEnabled
}

func (h NodePoolStateHandler) Sync(deployment appsv1.Deployment) error {
	if !h.isEnabled() {
		return nil
	}

	err := StartNodePool(h.config)
	if err != nil {
		return err
	}

	klog.Info("Start Successful")

	return nil //err
}

func (h NodePoolStateHandler) Clean() error {
	if !h.isEnabled() {
		return nil
	}

	err := StopNodePool(h.config)
	if err != nil {
		return err
	}

	klog.Info("Stop Successful")

	return nil //err
}
