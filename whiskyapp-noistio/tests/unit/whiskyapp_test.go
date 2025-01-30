package unit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// TestWhiskyApp_Validation tests the validation of WhiskyApp fields
func TestWhiskyApp_Validation(t *testing.T) {
	tests := []struct {
		name    string
		app     *WhiskyApp
		wantErr bool
	}{
		{
			name: "valid configuration",
			app: &WhiskyApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: WhiskyAppSpec{
					Name:     "test-app",
					Image:    "nginx:latest",
					Replicas: 3,
					Resources: Resources{
						Requests: ResourceRequirements{
							CPU:    "100m",
							Memory: "128Mi",
						},
						Limits: ResourceRequirements{
							CPU:    "500m",
							Memory: "512Mi",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			app: &WhiskyApp{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: WhiskyAppSpec{
					Image: "nginx:latest",
				},
			},
			wantErr: true,
		},
		{
			name: "missing image",
			app: &WhiskyApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: WhiskyAppSpec{
					Name: "test-app",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid resource limits",
			app: &WhiskyApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: WhiskyAppSpec{
					Name:  "test-app",
					Image: "nginx:latest",
					Resources: Resources{
						Limits: ResourceRequirements{
							CPU:    "invalid",
							Memory: "invalid",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWhiskyApp(tt.app)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// validateWhiskyApp validates a WhiskyApp resource
func validateWhiskyApp(app *WhiskyApp) error {
	if app.Name == "" {
		return fmt.Errorf("name is required")
	}
	if app.Spec.Name == "" {
		return fmt.Errorf("spec.name is required")
	}
	if app.Spec.Image == "" {
		return fmt.Errorf("spec.image is required")
	}
	return validateResources(app.Spec.Resources)
}

// validateResources validates resource requirements
func validateResources(res Resources) error {
	if err := validateResourceRequirements(res.Requests); err != nil {
		return fmt.Errorf("invalid requests: %v", err)
	}
	if err := validateResourceRequirements(res.Limits); err != nil {
		return fmt.Errorf("invalid limits: %v", err)
	}
	return nil
}

// validateResourceRequirements validates CPU and memory requirements
func validateResourceRequirements(req ResourceRequirements) error {
	if !isValidResourceValue(req.CPU) {
		return fmt.Errorf("invalid CPU value: %s", req.CPU)
	}
	if !isValidResourceValue(req.Memory) {
		return fmt.Errorf("invalid memory value: %s", req.Memory)
	}
	return nil
}

// isValidResourceValue checks if a resource value is valid
func isValidResourceValue(value string) bool {
	// Simple validation for demonstration
	// In production, use k8s.io/apimachinery/pkg/api/resource
	return value == "" || // Empty is valid for optional fields
		strings.HasSuffix(value, "m") || // millicpu
		strings.HasSuffix(value, "Mi") || // mebibytes
		strings.HasSuffix(value, "Gi") // gibibytes
}

// TestResource_Generation tests the generation of Kubernetes resources
func TestResource_Generation(t *testing.T) {
	app := &WhiskyApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: WhiskyAppSpec{
			Name:     "test-app",
			Image:    "nginx:latest",
			Replicas: 3,
			Resources: Resources{
				Requests: ResourceRequirements{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Limits: ResourceRequirements{
					CPU:    "500m",
					Memory: "512Mi",
				},
			},
		},
	}

	t.Run("ServiceAccount", func(t *testing.T) {
		sa := generateServiceAccount(app)
		require.NotNil(t, sa)
		assert.Equal(t, app.Spec.Name, sa.Name)
		assert.Equal(t, app.Namespace, sa.Namespace)
	})

	t.Run("Deployment", func(t *testing.T) {
		deploy := generateDeployment(app)
		require.NotNil(t, deploy)
		assert.Equal(t, app.Spec.Name, deploy.Name)
		assert.Equal(t, app.Namespace, deploy.Namespace)
		assert.Equal(t, app.Spec.Replicas, *deploy.Spec.Replicas)
		assert.Equal(t, app.Spec.Image, deploy.Spec.Template.Spec.Containers[0].Image)

		// Verify security context
		podSC := deploy.Spec.Template.Spec.SecurityContext
		require.NotNil(t, podSC)
		assert.True(t, *podSC.RunAsNonRoot)

		containerSC := deploy.Spec.Template.Spec.Containers[0].SecurityContext
		require.NotNil(t, containerSC)
		assert.False(t, *containerSC.AllowPrivilegeEscalation)
		assert.True(t, *containerSC.ReadOnlyRootFilesystem)

		// Verify resource requirements
		resources := deploy.Spec.Template.Spec.Containers[0].Resources
		assert.Equal(t, app.Spec.Resources.Requests.CPU, resources.Requests.Cpu().String())
		assert.Equal(t, app.Spec.Resources.Requests.Memory, resources.Requests.Memory().String())
		assert.Equal(t, app.Spec.Resources.Limits.CPU, resources.Limits.Cpu().String())
		assert.Equal(t, app.Spec.Resources.Limits.Memory, resources.Limits.Memory().String())

		// Verify probes
		require.NotNil(t, deploy.Spec.Template.Spec.Containers[0].ReadinessProbe)
		require.NotNil(t, deploy.Spec.Template.Spec.Containers[0].LivenessProbe)
	})

	t.Run("Service", func(t *testing.T) {
		svc := generateService(app)
		require.NotNil(t, svc)
		assert.Equal(t, app.Spec.Name, svc.Name)
		assert.Equal(t, app.Namespace, svc.Namespace)
		assert.Equal(t, "ClusterIP", string(svc.Spec.Type))
		assert.Equal(t, int32(80), svc.Spec.Ports[0].Port)
	})

	t.Run("NetworkPolicy", func(t *testing.T) {
		np := generateNetworkPolicy(app)
		require.NotNil(t, np)
		assert.Equal(t, app.Spec.Name, np.Name)
		assert.Equal(t, app.Namespace, np.Namespace)

		// Verify policy types
		require.Len(t, np.Spec.PolicyTypes, 2)
		assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
		assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)

		// Verify ingress rules
		require.Len(t, np.Spec.Ingress, 1)
		require.Len(t, np.Spec.Ingress[0].From, 1)
		require.NotNil(t, np.Spec.Ingress[0].From[0].PodSelector)
		assert.Equal(t, "ingressgateway", np.Spec.Ingress[0].From[0].PodSelector.MatchLabels["istio"])

		// Verify egress rules
		require.Len(t, np.Spec.Egress, 1)
		require.Len(t, np.Spec.Egress[0].To, 1)
		require.NotNil(t, np.Spec.Egress[0].To[0].NamespaceSelector)
		assert.Equal(t, "kube-system", np.Spec.Egress[0].To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
	})
}

// Helper functions to generate resources (these would be in your actual code)
func generateServiceAccount(app *WhiskyApp) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
	}
}

func generateDeployment(app *WhiskyApp) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &app.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
					},
					Containers: []corev1.Container{{
						Name:  app.Spec.Name,
						Image: app.Spec.Image,
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: boolPtr(false),
							ReadOnlyRootFilesystem:   boolPtr(true),
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(app.Spec.Resources.Requests.CPU),
								corev1.ResourceMemory: resource.MustParse(app.Spec.Resources.Requests.Memory),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(app.Spec.Resources.Limits.CPU),
								corev1.ResourceMemory: resource.MustParse(app.Spec.Resources.Limits.Memory),
							},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(80),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       10,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(80),
								},
							},
							InitialDelaySeconds: 15,
							PeriodSeconds:       20,
						},
					}},
				},
			},
		},
	}
}

func generateService(app *WhiskyApp) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Port: 80,
			}},
		},
	}
}

func generateNetworkPolicy(app *WhiskyApp) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"istio": "ingressgateway",
						},
					},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 80},
				}},
			}},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "kube-system",
						},
					},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: &[]corev1.Protocol{corev1.ProtocolUDP}[0],
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
				}, {
					Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
				}},
			}},
		},
	}
}
