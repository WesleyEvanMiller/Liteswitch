package main

import (
	"flag"
	"os"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
)

var (
	masterURL   string
	kubeconfig  string
	defaultPath string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "")
	flag.StringVar(&defaultPath, "configfile", "/etc/ns-controller/config.json", "")
	// klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)

	f, err := readConfigFile(defaultPath)
	if err != nil {
		klog.Fatal(err)
	}

	controller, err := NewController(kubeClient, kubeInformerFactory.Apps().V1().Deployments(), f)
	if err != nil {
		klog.Fatal(err)
	}

	kubeInformerFactory.Start(stopCh)

	if err = controller.Run(1, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func readConfigFile(path string) (*os.File, error) {
	f, err := os.Open(defaultPath)
	if err != nil {
		return &os.File{}, err
	}

	return f, nil
}
