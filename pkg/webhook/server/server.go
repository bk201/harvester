package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rancher/dynamiclistener"
	"github.com/rancher/dynamiclistener/server"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/harvester/harvester/pkg/webhook/clients"
	"github.com/harvester/harvester/pkg/webhook/config"
)

var (
	tlsName  = "harvester-webhook.harvester-system.svc"
	certName = "harvester-webhook-tls"
	caName   = "harvester-webhook-ca"
	port     = int32(443)

	validationPath              = "/v1/webhook/validation"
	mutationPath                = "/v1/webhook/mutation"
	clusterScope                = v1.ClusterScope
	namespaceScope              = v1.NamespacedScope
	failPolicyFail              = v1.Fail
	sideEffectClassNone         = v1.SideEffectClassNone
	sideEffectClassNoneOnDryRun = v1.SideEffectClassNoneOnDryRun
)

type AdmissionWebhookServer struct {
	context    context.Context
	restConfig *rest.Config
	options    *config.Options
}

func New(ctx context.Context, restConfig *rest.Config, options *config.Options) *AdmissionWebhookServer {
	return &AdmissionWebhookServer{
		context:    ctx,
		restConfig: restConfig,
		options:    options,
	}
}

func (s *AdmissionWebhookServer) ListenAndServe() error {
	clients, err := clients.New(s.context, s.restConfig, s.options.Threadiness)
	if err != nil {
		return err
	}

	validation, err := Validation(clients, s.options)
	if err != nil {
		return err
	}
	mutation, err := Mutation(clients, s.options)
	if err != nil {
		return err
	}

	router := mux.NewRouter()
	router.Handle(validationPath, validation)
	router.Handle(mutationPath, mutation)
	if err := s.listenAndServe(clients, router); err != nil {
		return err
	}

	if err := clients.Start(s.context); err != nil {
		return err
	}
	return nil
}

func (s *AdmissionWebhookServer) listenAndServe(clients *clients.Clients, handler http.Handler) error {
	apply := clients.Apply.WithDynamicLookup()
	clients.Core.Secret().OnChange(s.context, "secrets", func(key string, secret *corev1.Secret) (*corev1.Secret, error) {
		if secret == nil || secret.Name != caName || secret.Namespace != s.options.Namespace || len(secret.Data[corev1.TLSCertKey]) == 0 {
			return nil, nil
		}
		logrus.Info("Sleeping for 15 seconds then applying webhook config")
		// Sleep here to make sure server is listening and all caches are primed
		time.Sleep(15 * time.Second)

		return secret, apply.WithOwner(secret).ApplyObjects(&v1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "harvester-validator",
			},
			Webhooks: []v1.ValidatingWebhook{
				{
					Name: "validator.harvesterhci.io",
					ClientConfig: v1.WebhookClientConfig{
						Service: &v1.ServiceReference{
							Namespace: s.options.Namespace,
							Name:      "harvester-webhook",
							Path:      &validationPath,
							Port:      &port,
						},
						CABundle: secret.Data[corev1.TLSCertKey],
					},
					Rules: []v1.RuleWithOperations{
						{
							Operations: []v1.OperationType{
								v1.Create,
								v1.Update,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"harvesterhci.io"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"virtualmachineimages"},
								Scope:       &namespaceScope,
							},
						},
						{
							Operations: []v1.OperationType{
								v1.Create,
								v1.Update,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"harvesterhci.io"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"keypairs"},
								Scope:       &namespaceScope,
							},
						},
						{
							Operations: []v1.OperationType{
								v1.Create,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"harvesterhci.io"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"upgrades"},
								Scope:       &namespaceScope,
							},
						},
						{
							Operations: []v1.OperationType{
								v1.Create,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"harvesterhci.io"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"virtualmachinerestores"},
								Scope:       &namespaceScope,
							},
						},
						{
							Operations: []v1.OperationType{
								v1.Delete,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"cdi.kubevirt.io"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"datavolumes"},
								Scope:       &namespaceScope,
							},
						},
						{
							Operations: []v1.OperationType{
								v1.Create,
								v1.Delete,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"k8s.cni.cncf.io"},
								APIVersions: []string{"v1"},
								Resources:   []string{"network-attachment-definitions"},
								Scope:       &namespaceScope,
							},
						},
					},
					FailurePolicy:           &failPolicyFail,
					SideEffects:             &sideEffectClassNone,
					AdmissionReviewVersions: []string{"v1", "v1beta1"},
				},
			},
		}, &v1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "harvester-mutator",
			},
			Webhooks: []v1.MutatingWebhook{
				{
					Name: "mutator.harvesterhci.io",
					ClientConfig: v1.WebhookClientConfig{
						Service: &v1.ServiceReference{
							Namespace: s.options.Namespace,
							Name:      "harvester-webhook",
							Path:      &mutationPath,
							Port:      &port,
						},
						CABundle: secret.Data[corev1.TLSCertKey],
					},
					Rules: []v1.RuleWithOperations{
						{
							Operations: []v1.OperationType{
								v1.Create,
								v1.Update,
								v1.Delete,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"harvesterhci.io"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"users"},
								Scope:       &clusterScope,
							},
						},
						{
							Operations: []v1.OperationType{
								v1.Create,
								v1.Update,
								v1.Delete,
							},
							Rule: v1.Rule{
								APIGroups:   []string{"harvesterhci.io"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"virtualmachinetemplateversions"},
								Scope:       &namespaceScope,
							},
						},
					},
					FailurePolicy:           &failPolicyFail,
					SideEffects:             &sideEffectClassNoneOnDryRun,
					AdmissionReviewVersions: []string{"v1", "v1beta1"},
				},
			},
		})
	})

	return server.ListenAndServe(s.context, s.options.HTTPSListenPort, 0, handler, &server.ListenOpts{
		Secrets:       clients.Core.Secret(),
		CertNamespace: s.options.Namespace,
		CertName:      certName,
		CAName:        caName,
		TLSListenerConfig: dynamiclistener.Config{
			SANs: []string{
				tlsName,
			},
			FilterCN: dynamiclistener.OnlyAllow(tlsName),
		},
	})
}
