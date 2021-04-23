package supportbundle

import (
	"errors"
	"fmt"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	ctlappsv1 "github.com/rancher/wrangler/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/harvester/harvester/pkg/version"
)

type Manager struct {
	deployments ctlappsv1.DeploymentClient
	nodeCache   ctlcorev1.NodeCache
	services    ctlcorev1.ServiceClient
}

func (m *Manager) getManagerName(sb *harvesterv1.SupportBundle) string {
	return fmt.Sprintf("supportbundle-manager-%s", sb.Name)
}

func (m *Manager) Create(sb *harvesterv1.SupportBundle, image string) error {
	deployName := m.getManagerName(sb)
	logrus.Debugf("creating deployment %s with image %s", deployName, image)

	serviceAccountName, err := m.getServiceAccountName(sb.Namespace)
	if err != nil {
		return err
	}

	nodes, err := m.getNodes()
	if err != nil {
		return err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: sb.Namespace,
			Labels: map[string]string{
				"app":                 AppManager,
				SupportBundleLabelKey: sb.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       sb.Name,
					Kind:       sb.Kind,
					UID:        sb.UID,
					APIVersion: sb.APIVersion,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": AppManager},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                 AppManager,
						SupportBundleLabelKey: sb.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "manager",
							Image:           image,
							Args:            []string{"/usr/bin/support-bundle-utils", "manager"},
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "HARVESTER_NAMESPACE",
									Value: sb.Namespace,
								},
								{
									Name:  "HARVESTER_VERSION",
									Value: version.FriendlyVersion(),
								},
								{
									Name:  "HARVESTER_SUPPORT_BUNDLE_NAME",
									Value: sb.Name,
								},
								{
									Name:  "HARVESTER_SUPPORT_BUNDLE_NODE_COUNT",
									Value: fmt.Sprint(len(nodes)),
								},
								{
									Name:  "HARVESTER_SUPPORT_BUNDLE_DEBUG",
									Value: "true",
								},
								{
									Name: "HARVESTER_SUPPORT_BUNDLE_MANAGER_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "HARVESTER_SUPPORT_BUNDLE_IMAGE",
									Value: image,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
					ServiceAccountName: serviceAccountName,
				},
			},
		},
	}

	_, err = m.deployments.Create(deployment)
	if err != nil {
		return err
	}

	// service := corev1.Service{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      deployName,
	// 		Namespace: sb.Namespace,
	// 		Labels: map[string]string{
	// 			"app":                 AppManager,
	// 			SupportBundleLabelKey: sb.Name,
	// 		},
	// 		OwnerReferences: []metav1.OwnerReference{
	// 			{
	// 				Name:       sb.Name,
	// 				Kind:       sb.Kind,
	// 				UID:        sb.UID,
	// 				APIVersion: sb.APIVersion,
	// 			},
	// 		},
	// 	},
	// 	Spec: corev1.ServiceSpec{
	// 		Ports: []corev1.ServicePort{
	// 			{
	// 				Port: 8080,
	// 			},
	// 		},
	// 		Selector: map[string]string{
	// 			SupportBundleLabelKey: sb.Name,
	// 		},
	// 	},
	// }

	// _, err = m.services.Create(&service)
	// if err != nil {
	// 	if e := m.deployments.Delete(sb.Namespace, deployName, &metav1.DeleteOptions{}); e != nil {
	// 		logrus.Errorf("fail to cleanup: %s", e)
	// 	}
	// 	return err
	// }
	return nil
}

func (m *Manager) getServiceAccountName(namespace string) (string, error) {
	labelSelector := "app.kubernetes.io/component=apiserver"
	services, err := m.services.List(namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return "", err
	}
	if len(services.Items) != 1 {
		return "", fmt.Errorf("multiple services with label %s are found", labelSelector)
	}
	return services.Items[0].Name, nil
}

func (m *Manager) getNodes() ([]string, error) {
	r, err := labels.NewRequirement(HarvesterNodeLabelKey, selection.Equals, []string{HarvesterNodeLabelValue})
	if err != nil {
		return nil, err
	}

	nodes, err := m.nodeCache.List(labels.NewSelector().Add(*r))
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, errors.New("no Harvester nodes are found")
	}

	nodeNames := []string{}
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
	}
	return nodeNames, nil
}

func (m *Manager) GetManagerURL(sb *harvesterv1.SupportBundle) string {
	serviceName := m.getManagerName(sb)
	return fmt.Sprintf("http://%s.%s:8080", serviceName, sb.Namespace)
}
