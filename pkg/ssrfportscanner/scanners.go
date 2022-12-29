package ssrfportscanner

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VWebhookScan(options *pflag.FlagSet) {
	createNamespace(options)
	createValidatingWebhook(options)
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
	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example-webhook",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "example-webhook.example.com",
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
							admissionregistrationv1.Update,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      "example-service",
						Namespace: "default",
					},
					//CABundle: []byte(caBundle), // base64-encoded CA bundle
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
