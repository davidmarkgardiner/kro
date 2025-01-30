package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/kro/whiskyapp-noistio/tests/unit"
)

var (
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

// WhiskyAppReconciler reconciles a WhiskyApp object by creating and managing
// the underlying Kubernetes resources (ServiceAccount, Deployment, Service, NetworkPolicy).
type WhiskyAppReconciler struct {
	client.Client
}

// Reconcile handles WhiskyApp resources
func (r *WhiskyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	app := &unit.WhiskyApp{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create ServiceAccount
	sa := generateServiceAccount(app)
	if err := r.Create(ctx, sa); err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	}

	// Create or update Deployment
	deploy := generateDeployment(app)
	if err := r.Create(ctx, deploy); err != nil {
		if errors.IsAlreadyExists(err) {
			// Get existing deployment
			existing := &appsv1.Deployment{}
			if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, existing); err != nil {
				return ctrl.Result{}, err
			}
			// Update deployment
			existing.Spec = deploy.Spec
			if err := r.Update(ctx, existing); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	// Create Service
	svc := generateService(app)
	if err := r.Create(ctx, svc); err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	}

	// Create NetworkPolicy
	np := generateNetworkPolicy(app)
	if err := r.Create(ctx, np); err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// TestMain sets up the test environment by:
// 1. Configuring logging
// 2. Getting Kubernetes client configuration
// 3. Registering CRD types
// 4. Creating the WhiskyApp CRD
// 5. Starting the controller
func TestMain(m *testing.M) {
	// Set up logging
	log.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Get Kubernetes config
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(fmt.Sprintf("Error getting Kubernetes config: %v", err))
		}
	}

	// Register our types
	if err := unit.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("Error adding WhiskyApp scheme: %v", err))
	}

	// Register CRD types
	if err := apiextensionsv1.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("Error adding CRD types: %v", err))
	}

	// Create the client
	k8sClient, err = client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(fmt.Sprintf("Error creating client: %v", err))
	}

	// Create CRD
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "whiskyapps.kro.run",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "kro.run",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     "WhiskyApp",
				ListKind: "WhiskyAppList",
				Plural:   "whiskyapps",
				Singular: "whiskyapp",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1alpha1",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"spec": {
								Type: "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									"name":     {Type: "string"},
									"image":    {Type: "string"},
									"replicas": {Type: "integer", Minimum: &[]float64{1}[0]},
									"resources": {
										Type: "object",
										Properties: map[string]apiextensionsv1.JSONSchemaProps{
											"requests": {
												Type: "object",
												Properties: map[string]apiextensionsv1.JSONSchemaProps{
													"cpu":    {Type: "string"},
													"memory": {Type: "string"},
												},
											},
											"limits": {
												Type: "object",
												Properties: map[string]apiextensionsv1.JSONSchemaProps{
													"cpu":    {Type: "string"},
													"memory": {Type: "string"},
												},
											},
										},
									},
								},
								Required: []string{"name", "image"},
							},
						},
					},
				},
			}},
		},
	}

	if err := k8sClient.Create(ctx, crd); err != nil {
		if !errors.IsAlreadyExists(err) {
			panic(fmt.Sprintf("Error creating CRD: %v", err))
		}
	}

	// Start the controller
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme.Scheme,
		HealthProbeBindAddress: "0", // Disable health probe server
		LeaderElection:         false,
	})
	if err != nil {
		panic(fmt.Sprintf("Error creating manager: %v", err))
	}

	if err := (&WhiskyAppReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		panic(fmt.Sprintf("Error setting up controller: %v", err))
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(fmt.Sprintf("Error starting manager: %v", err))
		}
	}()

	os.Exit(m.Run())
}

// SetupWithManager sets up the controller with the Manager
func (r *WhiskyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&unit.WhiskyApp{}).
		Owns(&appsv1.Deployment{}). // Watch owned Deployments
		Complete(r)
}

// Helper functions to generate Kubernetes resources

// generateServiceAccount creates a ServiceAccount for the WhiskyApp.
// This is used to run the nginx pods with proper RBAC settings.
func generateServiceAccount(app *unit.WhiskyApp) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
	}
}

// generateDeployment creates a Deployment for the WhiskyApp with:
// - Security context for non-root execution
// - Resource limits and requests
// - Volume mounts for nginx
// - Readiness and liveness probes
// - Container ports
func generateDeployment(app *unit.WhiskyApp) *appsv1.Deployment {
	// Default resource values
	cpuRequest := "100m"
	memoryRequest := "128Mi"
	cpuLimit := "200m"
	memoryLimit := "256Mi"

	// Use specified values if provided
	if app.Spec.Resources.Requests.CPU != "" {
		cpuRequest = app.Spec.Resources.Requests.CPU
	}
	if app.Spec.Resources.Requests.Memory != "" {
		memoryRequest = app.Spec.Resources.Requests.Memory
	}
	if app.Spec.Resources.Limits.CPU != "" {
		cpuLimit = app.Spec.Resources.Limits.CPU
	}
	if app.Spec.Resources.Limits.Memory != "" {
		memoryLimit = app.Spec.Resources.Limits.Memory
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &app.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Spec.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": app.Spec.Name,
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
						RunAsUser:    &[]int64{101}[0], // nginx user
					},
					Volumes: []corev1.Volume{
						{
							Name: "nginx-cache",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "nginx-run",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "nginx-logs",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{{
						Name:  app.Spec.Name,
						Image: app.Spec.Image,
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: boolPtr(false),
							ReadOnlyRootFilesystem:   boolPtr(true), // Now we can set this back to true
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "nginx-cache",
								MountPath: "/var/cache/nginx",
							},
							{
								Name:      "nginx-run",
								MountPath: "/var/run",
							},
							{
								Name:      "nginx-logs",
								MountPath: "/var/log/nginx",
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(cpuRequest),
								corev1.ResourceMemory: resource.MustParse(memoryRequest),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(cpuLimit),
								corev1.ResourceMemory: resource.MustParse(memoryLimit),
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Protocol:      corev1.ProtocolTCP,
						}},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(80),
								},
							},
							InitialDelaySeconds: 1,
							PeriodSeconds:       2,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(80),
								},
							},
							InitialDelaySeconds: 2,
							PeriodSeconds:       4,
						},
					}},
				},
			},
		},
	}
}

// generateService creates a Service for the WhiskyApp to expose:
// - Port 80 for HTTP traffic
// - Selector matching the deployment pods
// - ClusterIP type for internal access
func generateService(app *unit.WhiskyApp) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": app.Spec.Name,
			},
			Ports: []corev1.ServicePort{{
				Port: 80,
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

// generateNetworkPolicy creates a NetworkPolicy for the WhiskyApp that:
// - Allows ingress from the ingressgateway on port 80
// - Allows egress to kube-system for DNS (TCP/UDP port 53)
// - Denies all other traffic
func generateNetworkPolicy(app *unit.WhiskyApp) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Spec.Name,
			Namespace: app.Namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Spec.Name,
				},
			},
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

// boolPtr is a helper function to get a pointer to a boolean value.
// This is commonly needed for Kubernetes API objects that use pointers
// for boolean fields.
func boolPtr(b bool) *bool {
	return &b
}

// TestWhiskyApp_Integration tests the complete lifecycle of a WhiskyApp resource:
// 1. Creates a unique test namespace
// 2. Deploys a WhiskyApp instance
// 3. Verifies all resources are created correctly
// 4. Tests scaling functionality
// 5. Validates security settings
// 6. Cleans up all resources
func TestWhiskyApp_Integration(t *testing.T) {
	// Create unique namespace for this test
	namespaceName := fmt.Sprintf("test-whiskyapp-%d", time.Now().Unix())
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	err := k8sClient.Create(ctx, namespace)
	require.NoError(t, err)

	// Clean up namespace after test
	defer func() {
		t.Logf("Cleaning up namespace: %s", namespaceName)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				t.Logf("Timed out cleaning up namespace: %s", namespaceName)
				return
			default:
				// Get latest namespace state
				ns := &corev1.Namespace{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, ns); err != nil {
					if client.IgnoreNotFound(err) == nil {
						return
					}
					t.Logf("Error getting namespace: %v", err)
					time.Sleep(time.Second)
					continue
				}
				// If namespace exists but isn't being deleted yet, delete it
				if ns.DeletionTimestamp == nil {
					if err := k8sClient.Delete(ctx, ns); err != nil {
						t.Logf("Error deleting namespace: %v", err)
					}
				}
				// Remove finalizers if they exist
				if len(ns.Finalizers) > 0 {
					ns.Finalizers = nil
					if err := k8sClient.Update(ctx, ns); err != nil {
						if client.IgnoreNotFound(err) == nil {
							return
						}
						t.Logf("Error removing finalizers: %v", err)
					}
				}
				time.Sleep(time.Second)
			}
		}
	}()

	// Create WhiskyApp instance
	app := &unit.WhiskyApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: namespaceName,
		},
		Spec: unit.WhiskyAppSpec{
			Name:     "test-app",
			Image:    "nginx:latest",
			Replicas: 1,
			Resources: unit.Resources{
				Requests: unit.ResourceRequirements{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Limits: unit.ResourceRequirements{
					CPU:    "200m",
					Memory: "256Mi",
				},
			},
		},
	}

	// Create the WhiskyApp
	err = k8sClient.Create(ctx, app)
	require.NoError(t, err)

	// Wait for resources to be created
	t.Log("Waiting for resources to be created...")
	err = wait.PollImmediate(100*time.Millisecond, 30*time.Second, func() (bool, error) {
		// Check ServiceAccount
		sa := &corev1.ServiceAccount{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      app.Spec.Name,
			Namespace: namespaceName,
		}, sa)
		if err != nil {
			return false, nil
		}

		// Check Deployment
		deploy := &appsv1.Deployment{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      app.Spec.Name,
			Namespace: namespaceName,
		}, deploy)
		if err != nil {
			return false, nil
		}

		// Check Service
		svc := &corev1.Service{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      app.Spec.Name,
			Namespace: namespaceName,
		}, svc)
		if err != nil {
			return false, nil
		}

		// Check NetworkPolicy
		np := &networkingv1.NetworkPolicy{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      app.Spec.Name,
			Namespace: namespaceName,
		}, np)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	require.NoError(t, err, "Failed waiting for resources to be created")

	// Test updating the WhiskyApp
	t.Log("Testing WhiskyApp updates...")

	// Get the latest version
	updatedApp := &unit.WhiskyApp{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      app.Name,
		Namespace: namespaceName,
	}, updatedApp)
	require.NoError(t, err)

	// Update replicas
	updatedApp.Spec.Replicas = 2
	err = k8sClient.Update(ctx, updatedApp)
	require.NoError(t, err)

	// Wait for deployment to scale
	t.Log("Waiting for deployment to scale...")
	err = wait.PollImmediate(100*time.Millisecond, 30*time.Second, func() (bool, error) {
		deploy := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      app.Spec.Name,
			Namespace: namespaceName,
		}, deploy)
		if err != nil {
			t.Logf("Error getting deployment: %v", err)
			return false, nil
		}

		// Log deployment status for debugging
		t.Logf("Deployment status - Replicas: %d, Ready: %d, Updated: %d, Available: %d",
			deploy.Status.Replicas,
			deploy.Status.ReadyReplicas,
			deploy.Status.UpdatedReplicas,
			deploy.Status.AvailableReplicas)

		// Get and log pod events if pods aren't ready
		if deploy.Status.ReadyReplicas != deploy.Status.Replicas {
			podList := &corev1.PodList{}
			if err := k8sClient.List(ctx, podList, client.InNamespace(namespaceName), client.MatchingLabels{"app": app.Spec.Name}); err != nil {
				t.Logf("Error listing pods: %v", err)
			} else {
				for _, pod := range podList.Items {
					t.Logf("Pod %s status: %s", pod.Name, pod.Status.Phase)
					if len(pod.Status.ContainerStatuses) > 0 {
						cs := pod.Status.ContainerStatuses[0]
						if !cs.Ready && cs.State.Waiting != nil {
							t.Logf("Container waiting: %s - %s", cs.State.Waiting.Reason, cs.State.Waiting.Message)
						}
					}
				}
			}
		}

		return deploy.Status.ReadyReplicas == 2 &&
			deploy.Status.UpdatedReplicas == 2 &&
			deploy.Status.AvailableReplicas == 2, nil
	})
	require.NoError(t, err, "Failed waiting for deployment to scale")

	// Verify deployment security settings
	deploy := &appsv1.Deployment{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      app.Spec.Name,
		Namespace: namespaceName,
	}, deploy)
	require.NoError(t, err)

	// Check security context
	podSC := deploy.Spec.Template.Spec.SecurityContext
	require.NotNil(t, podSC)
	assert.True(t, *podSC.RunAsNonRoot)
	assert.Equal(t, int64(101), *podSC.RunAsUser) // Verify nginx user

	containerSC := deploy.Spec.Template.Spec.Containers[0].SecurityContext
	require.NotNil(t, containerSC)
	assert.False(t, *containerSC.AllowPrivilegeEscalation)
	assert.True(t, *containerSC.ReadOnlyRootFilesystem) // Verify root filesystem is read-only
	require.NotNil(t, containerSC.Capabilities)
	assert.Contains(t, containerSC.Capabilities.Drop, corev1.Capability("ALL")) // Verify all capabilities are dropped

	// Verify volume mounts for nginx
	container := deploy.Spec.Template.Spec.Containers[0]
	expectedMounts := map[string]string{
		"nginx-cache": "/var/cache/nginx",
		"nginx-run":   "/var/run",
		"nginx-logs":  "/var/log/nginx",
	}
	for _, mount := range container.VolumeMounts {
		expectedPath, exists := expectedMounts[mount.Name]
		assert.True(t, exists, "Expected volume mount %s", mount.Name)
		assert.Equal(t, expectedPath, mount.MountPath, "Expected mount path for %s", mount.Name)
	}

	// Verify network policy
	np := &networkingv1.NetworkPolicy{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      app.Spec.Name,
		Namespace: namespaceName,
	}, np)
	require.NoError(t, err)
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	assert.Contains(t, np.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
}
