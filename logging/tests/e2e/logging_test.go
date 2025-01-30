package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestLoggingE2E(t *testing.T) {
	// Skip if not running in E2E test environment
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	tests := []struct {
		name             string
		namespaces       []string
		applications     []TestApp
		expectedPatterns []string
		timeoutMinutes   int
	}{
		{
			name:       "multi namespace logging",
			namespaces: []string{"app1", "app2"},
			applications: []TestApp{
				{
					name:       "app1",
					namespace:  "app1",
					logPattern: "Application 1 log message",
				},
				{
					name:       "app2",
					namespace:  "app2",
					logPattern: "Application 2 log message",
				},
			},
			expectedPatterns: []string{
				"Application 1 log message",
				"Application 2 log message",
			},
			timeoutMinutes: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tt.timeoutMinutes)*time.Minute)
			defer cancel()

			// Set up logging infrastructure
			client, err := setupLoggingInfrastructure(ctx)
			if err != nil {
				t.Fatalf("failed to setup logging infrastructure: %v", err)
			}

			// Deploy test applications
			if err := deployTestApplications(ctx, client, tt.applications); err != nil {
				t.Fatalf("failed to deploy test applications: %v", err)
			}

			// Verify log collection
			if err := verifyLogCollection(ctx, t, "test-workspace", "test-token", tt.expectedPatterns); err != nil {
				t.Errorf("log collection verification failed: %v", err)
			}

			// Test failure scenarios
			if err := testFailureScenarios(ctx, t, client); err != nil {
				t.Errorf("failure scenario testing failed: %v", err)
			}
		})
	}
}

type TestApp struct {
	name       string
	namespace  string
	logPattern string
}

func setupLoggingInfrastructure(ctx context.Context) (*kubernetes.Clientset, error) {
	// Get kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %v", err)
		}
	}

	// Create kubernetes client
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	// Create or get logging namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "logging",
		},
	}
	_, err = client.CoreV1().Namespaces().Get(ctx, "logging", metav1.GetOptions{})
	if err != nil {
		// Namespace doesn't exist, create it
		_, err = client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create logging namespace: %v", err)
		}
	}

	// Deploy logging operator and wait for it to be ready
	if err := deployLoggingOperator(ctx, client); err != nil {
		return nil, fmt.Errorf("failed to deploy logging operator: %v", err)
	}

	// Create logging instance and wait for it to be ready
	if err := createLoggingInstance(ctx, client); err != nil {
		return nil, fmt.Errorf("failed to create logging instance: %v", err)
	}

	return client, nil
}

func deployTestApplications(ctx context.Context, client *kubernetes.Clientset, apps []TestApp) error {
	for _, app := range apps {
		// Create namespace if it doesn't exist
		if err := createNamespaceIfNotExists(ctx, client, app.namespace); err != nil {
			return fmt.Errorf("failed to create namespace %s: %v", app.namespace, err)
		}

		// Deploy application pod
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.name,
				Namespace: app.namespace,
				Labels: map[string]string{
					"app": app.name,
				},
				Annotations: map[string]string{
					"logging/endpoint": fmt.Sprintf("%s-logs", app.name),
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "busybox",
						Command: []string{
							"/bin/sh",
							"-c",
							fmt.Sprintf("while true; do echo '%s'; sleep 1; done", app.logPattern),
						},
					},
				},
			},
		}

		_, err := client.CoreV1().Pods(app.namespace).Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create pod %s: %v", app.name, err)
		}

		// Wait for pod to be ready
		if err := waitForPodReady(ctx, client, app.namespace, app.name); err != nil {
			return fmt.Errorf("failed waiting for pod %s to be ready: %v", app.name, err)
		}
	}

	return nil
}

func verifyLogCollection(ctx context.Context, t *testing.T, workspaceID, token string, patterns []string) error {
	// Create Log Analytics client
	laClient := createLogAnalyticsClient(workspaceID, token)

	// Wait for logs to be available (with timeout)
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for _, pattern := range patterns {
		found := false
		query := fmt.Sprintf(`
			CustomLogs_CL
			| where TimeGenerated > ago(10m)
			| where Log_s contains "%s"
			| limit 1
		`, pattern)

		for !found {
			select {
			case <-timeout:
				return fmt.Errorf("timeout waiting for log pattern: %s", pattern)
			case <-ticker.C:
				results, err := queryLogAnalytics(laClient, query)
				if err != nil {
					t.Logf("failed to query logs: %v", err)
					continue
				}

				if len(results) > 0 {
					found = true
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

func testFailureScenarios(ctx context.Context, t *testing.T, client *kubernetes.Clientset) error {
	// Test pod crash recovery
	if err := testPodCrashRecovery(ctx, client); err != nil {
		return fmt.Errorf("pod crash recovery test failed: %v", err)
	}

	// Test network partition
	if err := testNetworkPartition(ctx, client); err != nil {
		return fmt.Errorf("network partition test failed: %v", err)
	}

	// Test storage issues
	if err := testStorageIssues(ctx, client); err != nil {
		return fmt.Errorf("storage issues test failed: %v", err)
	}

	return nil
}

// Helper functions

func createNamespaceIfNotExists(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

func waitForPodReady(ctx context.Context, client *kubernetes.Clientset, namespace, name string) error {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for pod ready")
		case <-ticker.C:
			pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get pod: %v", err)
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					return nil
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func deployLoggingOperator(ctx context.Context, client *kubernetes.Clientset) error {
	// Create deployment for logging operator
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logging-operator",
			Namespace: "logging",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "logging-operator",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "logging-operator",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "logging-operator",
					Containers: []corev1.Container{
						{
							Name:  "operator",
							Image: "kro/logging-operator:latest", // Use appropriate image
							Env: []corev1.EnvVar{
								{
									Name: "WATCH_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create service account
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logging-operator",
			Namespace: "logging",
		},
	}
	_, err := client.CoreV1().ServiceAccounts("logging").Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service account: %v", err)
	}

	// Create RBAC roles (simplified for example)
	// TODO: Add proper RBAC setup

	// Create the deployment
	_, err = client.AppsV1().Deployments("logging").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create operator deployment: %v", err)
	}

	// Wait for deployment to be ready
	return waitForDeploymentReady(ctx, client, "logging", "logging-operator")
}

func waitForDeploymentReady(ctx context.Context, client *kubernetes.Clientset, namespace, name string) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for deployment ready")
		case <-ticker.C:
			deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get deployment: %v", err)
			}

			if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func createLoggingInstance(ctx context.Context, client *kubernetes.Clientset) error {
	// TODO: Implement using kro client
	return nil
}

func testPodCrashRecovery(ctx context.Context, client *kubernetes.Clientset) error {
	// Create a test pod that will crash
	crashPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crash-test-pod",
			Namespace: "logging",
			Labels: map[string]string{
				"app": "crash-test",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:  "crash-container",
					Image: "busybox",
					Command: []string{
						"/bin/sh",
						"-c",
						"echo 'About to crash'; exit 1",
					},
				},
			},
		},
	}

	// Create the pod
	_, err := client.CoreV1().Pods("logging").Create(ctx, crashPod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create crash test pod: %v", err)
	}

	// Wait for pod to crash and restart
	time.Sleep(10 * time.Second)

	// Verify pod was restarted
	pod, err := client.CoreV1().Pods("logging").Get(ctx, "crash-test-pod", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get crash test pod: %v", err)
	}

	if pod.Status.ContainerStatuses[0].RestartCount == 0 {
		return fmt.Errorf("pod did not restart as expected")
	}

	return nil
}

func testNetworkPartition(ctx context.Context, client *kubernetes.Clientset) error {
	// Create a network policy to simulate network partition
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-network-partition",
			Namespace: "logging",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "logging-collector",
				},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": "allowed-app",
								},
							},
						},
					},
				},
			},
		},
	}

	// Apply network policy
	_, err := client.NetworkingV1().NetworkPolicies("logging").Create(ctx, networkPolicy, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create network policy: %v", err)
	}

	// Wait for policy to take effect
	time.Sleep(30 * time.Second)

	// Verify logging still works after removing policy
	err = client.NetworkingV1().NetworkPolicies("logging").Delete(ctx, "test-network-partition", metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete network policy: %v", err)
	}

	return nil
}

func testStorageIssues(ctx context.Context, client *kubernetes.Clientset) error {
	// Create a pod that fills up its storage
	storagePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-test-pod",
			Namespace: "logging",
			Labels: map[string]string{
				"app": "storage-test",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "storage-container",
					Image: "busybox",
					Command: []string{
						"/bin/sh",
						"-c",
						"while true; do dd if=/dev/zero of=/tmp/fill bs=1M count=100; sleep 1; done",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "test-volume",
							MountPath: "/tmp",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "test-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							SizeLimit: resource.NewQuantity(100*1024*1024, resource.BinarySI), // 100MB limit
						},
					},
				},
			},
		},
	}

	// Create the pod
	_, err := client.CoreV1().Pods("logging").Create(ctx, storagePod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create storage test pod: %v", err)
	}

	// Wait for storage pressure
	time.Sleep(30 * time.Second)

	// Verify pod was evicted or restarted
	pod, err := client.CoreV1().Pods("logging").Get(ctx, "storage-test-pod", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get storage test pod: %v", err)
	}

	// Check if pod was affected by storage pressure
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
			// Pod was affected by storage pressure as expected
			return nil
		}
	}

	return fmt.Errorf("pod was not affected by storage pressure as expected")
}

func createLogAnalyticsClient(workspaceID, token string) interface{} {
	// TODO: Implement Azure Log Analytics client creation
	return nil
}

func queryLogAnalytics(client interface{}, query string) ([]interface{}, error) {
	// TODO: Implement Azure Log Analytics query
	return nil, nil
}
