package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	toolswatch "k8s.io/client-go/tools/watch"
)

func ConnectK8s() (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, nil
}

type eventWatcher struct {
	client kubernetes.Interface
}

func (e *eventWatcher) WatchWithContext(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
	return e.client.CoreV1().Events("").Watch(ctx, options)
}

func WatchEvents(ctx context.Context, c *kubernetes.Clientset, initialResourceVersion string) (watch.Interface, error) {
	if initialResourceVersion == "" {
		initialResourceVersion = "1"
	}
	watcherClient := &eventWatcher{client: c}

	rw, err := toolswatch.NewRetryWatcherWithContext(ctx, initialResourceVersion, watcherClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create retry watcher: %w", err)
	}

	return rw, nil
}
