package collector

import (
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
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

func WatchEvents(ctx context.Context, c *kubernetes.Clientset) <-chan v1.Event {
	eventCh := make(chan v1.Event, 100)

	source := cache.NewListWatchFromClient(
		c.CoreV1().RESTClient(),
		"events",
		v1.NamespaceAll,
		fields.Everything(),
	)

	informer := cache.NewSharedInformer(source, &v1.Event{}, 0)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if event, ok := obj.(*v1.Event); ok {
				log.Printf("Received event: %v", event.Name)
				select {
				case eventCh <- *event:
				case <-ctx.Done():
					return
				}
			}
		},
	})

	go informer.Run(ctx.Done())

	return eventCh
}
