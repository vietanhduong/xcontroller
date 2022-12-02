package main

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/vietanhduong/xcontroller/pkg/util/log"
)

func getKubeConfig(kubeConfig string) (*kubernetes.Clientset, error) {
	var fullKubeConfigPath string
	var err error

	if kubeConfig != "" {
		fullKubeConfigPath, err = filepath.Abs(kubeConfig)
		if err != nil {
			return nil, fmt.Errorf("cannot expand path %s: %v", kubeConfig, err)
		}
	}

	if fullKubeConfigPath != "" {
		log.Debugf("Creating Kubernetes client from %s", fullKubeConfigPath)
	} else {
		log.Debugf("Creating in-cluster Kubernetes client")
	}

	return newKubeClient(fullKubeConfigPath)
}

func newKubeClient(kubeconfig string) (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubeconfig
	overrides := clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
