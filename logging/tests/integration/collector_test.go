package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestLoggingCollector(t *testing.T) {
	// Skip if not running in integration test environment
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tests := []struct {
		name           string
		namespace      string
		podName        string
		expectedLogs   string
		timeoutSeconds int
	}{
		{
			name:           "collect logs from test pod",
			namespace:      "test-logging",
			podName:        "test-pod",
			expectedLogs:   `test log message`,
			timeoutSeconds: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up test environment
			client, cleanup := setupTestEnvironment(ctx, t, tt.namespace)
			defer cleanup()

			// Deploy test pod
			if err := deployTestPod(ctx, client, tt.namespace, tt.podName); err != nil {
				t.Fatalf("failed to deploy test pod: %v", err)
			}

			// Verify logs
			if err := verifyLogs(ctx, t, "test-workspace", "test-token", "test-table", tt.expectedLogs); err != nil {
				t.Errorf("log verification failed: %v", err)
			}
		})
	}
}

func setupTestEnvironment(ctx context.Context, t *testing.T, namespace string) (*kubernetes.Clientset, func()) {
	// Get kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			t.Fatalf("failed to get kubernetes config: %v", err)
		}
	}

	// Create kubernetes client
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("failed to create kubernetes client: %v", err)
	}

	// Create or get namespace
	if err := createNamespaceIfNotExists(ctx, client, namespace); err != nil {
		t.Fatalf("failed to create or get namespace: %v", err)
	}

	// Deploy logging instance
	// TODO: Use kro client to create logging instance
	// For now, we'll use kubectl apply as a workaround
	if err := applyLoggingInstance(namespace); err != nil {
		t.Fatalf("failed to deploy logging instance: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		if err := client.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
			t.Logf("failed to delete namespace: %v", err)
		}
	}

	return client, cleanup
}

func createNamespaceIfNotExists(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace exists, delete it first to ensure clean state
		err = client.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete existing namespace: %v", err)
		}

		// Wait for namespace to be deleted
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return fmt.Errorf("timeout waiting for namespace deletion")
			case <-ticker.C:
				_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					// Namespace is deleted, we can proceed
					break
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Create new namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	return nil
}

func deployTestPod(ctx context.Context, client *kubernetes.Clientset, namespace, name string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test-pod",
			},
			Annotations: map[string]string{
				"logging/endpoint": "test-endpoint",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "busybox",
					Command: []string{
						"/bin/sh",
						"-c",
						"while true; do echo 'test log message'; sleep 1; done",
					},
				},
			},
		},
	}

	_, err := client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create test pod: %v", err)
	}

	// Wait for pod to be ready
	return waitForPodReady(ctx, client, namespace, name)
}

func verifyLogs(ctx context.Context, t *testing.T, workspaceID, token, table, expectedLog string) error {
	// Create Log Analytics client
	laClient := createLogAnalyticsClient(workspaceID, token)

	// Wait for logs to be available (with timeout)
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	query := fmt.Sprintf(`
		%s_CL
		| where TimeGenerated > ago(5m)
		| where Log_s contains "%s"
		| limit 1
	`, table, expectedLog)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for logs")
		case <-ticker.C:
			// Query Log Analytics
			results, err := queryLogAnalytics(laClient, query)
			if err != nil {
				t.Logf("failed to query logs: %v", err)
				continue
			}

			if len(results) > 0 {
				return nil // Found the expected log
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Helper functions

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

// TODO: Implement these functions
func applyLoggingInstance(namespace string) error {
	// For now, return nil as this will be implemented with kro client
	return nil
}

func createLogAnalyticsClient(workspaceID, token string) interface{} {
	// TODO: Implement Azure Log Analytics client creation
	return nil
}

func queryLogAnalytics(client interface{}, query string) ([]interface{}, error) {
	// TODO: Implement Azure Log Analytics query
	return nil, nil
}
