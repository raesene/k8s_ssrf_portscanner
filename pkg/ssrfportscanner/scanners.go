package ssrfportscanner

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VWebhookScan(options *pflag.FlagSet) {
	if checkNamespace(options) {
		log.Print("VWebhookScan: namespace already exists")
	} else {
		createNamespace(options)
	}
	if checkWebhook(options) {
		log.Print("VWebhookScan: validating webhook already exists")
		deleteWebhook(options)
	}
	createValidatingWebhook(options)
	result := createPod(options)
	//fmt.Println(result)
	host, _ := options.GetString("target")
	port, _ := options.GetString("port")
	switch {
	case result == nil:
		fmt.Println("that's weird that should not happen")
	case strings.Contains(result.Error(), "connection refused"):
		fmt.Println("Host " + host + " : Port " + port + " is closed")
	case strings.Contains(result.Error(), "certificate is valid for"):
		fmt.Println("Host " + host + " : Port " + port + " speaks HTTPS but needs different SNI")
	case strings.Contains(result.Error(), "certificate signed by unknown authority"):
		fmt.Println("Host " + host + " : Port " + port + " speaks HTTPS but the API server does not trust the certificate")
	case strings.Contains(result.Error(), "json parse error"):
		fmt.Println("Host " + host + " : Port " + port + " speaks HTTPS and has a valid certificate")
	case strings.Contains(result.Error(), "no route to host"):
		fmt.Println("Host " + host + " : Host is not reachable")
	case strings.Contains(result.Error(), "context deadline exceeded"):
		fmt.Println("Host " + host + " : Port " + port + " is filtered")
	case strings.Contains(result.Error(), "i/o timeout"):
		fmt.Println("Host " + host + " : Port " + port + " is filtered")
	case strings.Contains(result.Error(), "server gave HTTP response to HTTPS client"):
		fmt.Println("Host " + host + " : Port " + port + " is open but speaks HTTP not HTTPS")
	case strings.Contains(result.Error(), "first record does not look like a TLS handshake"):
		fmt.Println("Host " + host + " : Port " + port + " is open but speaks a non-HTTP protocol")
	// This case comes from AKS specifically, perhaps a different golang version?
	case strings.Contains(result.Error(), "EOF"):
		fmt.Println("Host " + host + " : Port " + port + " is not available/closed")
	// Handle the going to fast case until we fix it
	case strings.Contains(result.Error(), "it is being terminated"):
		fmt.Println("Namespace being teriminated, slow downa and try again :)")
	case strings.Contains(result.Error(), "Unauthorized"):
		fmt.Println("Host " + host + " : Target " + port + " requires authorization")
	case strings.Contains(result.Error(), "certificate relies on legacy Common Name field"):
		fmt.Println("Host " + host + " : Target " + port + " uses a certificate with a Common Name field")
	case strings.Contains(result.Error(), "doesn't contain any IP SANs"):
		fmt.Println("Host " + host + " : Target " + port + " uses a certificate without an IP SAN but we tried to connect via IP")
	case strings.Contains(result.Error(), "certificate has expired or is not yet valid"):
		fmt.Println("Host " + host + " : Target " + port + " Has an expired certificate")
	case strings.Contains(result.Error(), "network is unreachable"):
		fmt.Println("Host " + host + " : is unreachable")
	default:
		fmt.Println("Oooh case we don't know about, please file an issue with the error message below!")
		fmt.Println(result.Error())
	}
	deleteWebhook(options)
	deleteNamespace(options)
}

//Connect to a Kubernetes cluster and create a namespace
func createNamespace(options *pflag.FlagSet) {
	name, _ := options.GetString("namespace")
	clientset, err := initKubeClient()
	if err != nil {
		log.Printf("createNamespace: failed creating Clientset with", err)
		return
	}

	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"ssrf-portscanner": "true",
			},
		},
	}

	// Get the namespace interface
	namespaces := clientset.CoreV1().Namespaces()

	_, err = namespaces.Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil {
		log.Printf("createNamespace: failed creating namespace with", err)
		return
	}

}

//Connect to a Kubernetes cluster and create a validating webhook
func createValidatingWebhook(options *pflag.FlagSet) {
	clientset, err := initKubeClient()
	if err != nil {
		log.Printf("createValidatingWebhook: failed creating Clientset with", err)
		return
	}

	//target, _ := options.GetString("target")
	sideEffect := admissionregistrationv1.SideEffectClassNone
	scope := admissionregistrationv1.NamespacedScope
	target, _ := options.GetString("target")
	port, _ := options.GetString("port")
	url := "https://" + target + ":" + port
	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ssrf-portscanner-webhook",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "ssrf-portscanner-webhook.example.com",
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"ssrf-portscanner": "true",
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       &scope,
						},
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL: &url,
				},
				SideEffects:             &sideEffect,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
			},
		},
	}
	_, err = clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), webhookConfig, metav1.CreateOptions{})
	if err != nil {
		log.Print("createValidatingWebhook: failed creating validating webhook with", err)
	}
}

// Create a new pod in the program namespace
func createPod(options *pflag.FlagSet) error {
	clientset, err := initKubeClient()
	if err != nil {
		log.Printf("createPod: failed creating Clientset with", err)
		return (err)
	}

	name, _ := options.GetString("namespace")
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ssrf-portscanner",
			Labels: map[string]string{
				"ssrf-portscanner": "true",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "ssrf-portscanner",
					Image: "busybox",
				},
			},
		},
	}
	_, err = clientset.CoreV1().Pods(name).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return (err)
	}
	return nil
}

func checkNamespace(options *pflag.FlagSet) bool {
	name, _ := options.GetString("namespace")
	clientset, err := initKubeClient()
	if err != nil {
		log.Printf("checkNamespace: failed creating Clientset with", err)
		return false
	}

	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		//log.Printf("checkNamespace: failed getting namespace with", err)
		return false
	}
	return true
}

func checkWebhook(options *pflag.FlagSet) bool {
	clientset, err := initKubeClient()
	if err != nil {
		log.Printf("checkWebhook: failed creating Clientset with", err)
		return false
	}

	_, err = clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), "ssrf-portscanner-webhook", metav1.GetOptions{})
	if err != nil {
		//log.Printf("checkWebhook: failed getting validating webhook with", err)
		return false
	}
	return true
}

func deleteWebhook(options *pflag.FlagSet) {
	clientset, err := initKubeClient()
	if err != nil {
		log.Printf("deleteWebhook: failed creating Clientset with", err)
		return
	}

	err = clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), "ssrf-portscanner-webhook", metav1.DeleteOptions{})
	if err != nil {
		log.Printf("deleteWebhook: failed deleting validating webhook with", err)
	}
}

func deleteNamespace(options *pflag.FlagSet) {
	clientset, err := initKubeClient()
	if err != nil {
		log.Printf("deleteNamespace: failed creating Clientset with", err)
		return
	}

	name, _ := options.GetString("namespace")
	err = clientset.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("deleteNamespace: failed deleting namespace with", err)
	}
}
