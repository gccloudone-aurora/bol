package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/gccloudone-aurora/bol/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClientset creates a new Kubernetes clientset.
func CreateClientset(configAuth util.KubernetesAuth) (*k8s.Clientset, error) {
	var config *rest.Config
	var err error

	switch configAuth.Method {
	case "kubeconfigPath":
		config, err = clientcmd.BuildConfigFromFlags("", configAuth.KubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed building Kubernetes config from flags: %v", err)
		}
		log.Println("CreateClientset: Using kubeconfig Kubernetes configuration")
	case "incluster":
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("CreateClientset: failed to create kubernetes config from flags or from the Kubernetes service account provided to this workload's pod: %v", err)
		}
		log.Println("CreateClientset: Using incluster Kubernetes configuration")
	case "manual":
		caData, err := base64.StdEncoding.DecodeString(configAuth.Manual.TLSClientConfig.CaData)
		if err != nil {
			log.Fatalf("Failed to decode CAData: %v", err)
		}

		config = &rest.Config{
			Host:        configAuth.Manual.Host,
			BearerToken: configAuth.Manual.BearerToken,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: configAuth.Manual.TLSClientConfig.Insecure,
				CAData:   caData,
			},
		}
		log.Println("CreateClientset: Using manual Kubernetes configuration")
	}

	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("CreateClientset: failed to create kubernetes clientset: %v", err)
	}

	return clientset, nil
}

// GetNamespaces retrieves all namespaces.
// Returns a map of namespace names to namespace objects.
func GetNamespaces(clientset k8s.Clientset) (map[string]v1.Namespace, error) {
	ctx := context.Background()

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("GetNamespaces: failed to list namespaces: %v", err)
	}

	namespaceMap := make(map[string]v1.Namespace)

	for _, ns := range namespaces.Items {
		namespaceMap[ns.Name] = ns
	}

	return namespaceMap, nil
}
