package integration

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func TestGatewayIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test")
	}

	// Get kubernetes config
	config, err := getKubeConfig()
	require.NoError(t, err)

	// Create kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	// Create gateway client
	gatewayClientset, err := gatewayclient.NewForConfig(config)
	require.NoError(t, err)

	// Create test namespace
	ns := &metav1.ObjectMeta{
		Name: "gateway-test-" + randString(6),
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: *ns,
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	defer func() {
		_ = clientset.CoreV1().Namespaces().Delete(context.Background(), ns.Name, metav1.DeleteOptions{})
	}()

	// Create test TLS secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tls-secret",
			Namespace: ns.Name,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("test-cert"),
			"tls.key": []byte("test-key"),
		},
	}
	_, err = clientset.CoreV1().Secrets(ns.Name).Create(context.Background(), secret, metav1.CreateOptions{})
	require.NoError(t, err)

	// Test cases
	tests := []struct {
		name    string
		gateway *gatewayv1.Gateway
		check   func(*testing.T, *gatewayv1.Gateway)
	}{
		{
			name: "create basic gateway",
			gateway: &gatewayv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gateway",
					Namespace: ns.Name,
				},
				Spec: gatewayv1.GatewaySpec{
					GatewayClassName: "istio",
					Listeners: []gatewayv1.Listener{
						{
							Name:     gatewayv1.SectionName("https"),
							Protocol: gatewayv1.HTTPSProtocolType,
							Port:     gatewayv1.PortNumber(443),
							TLS: &gatewayv1.GatewayTLSConfig{
								Mode: ptr(gatewayv1.TLSModeTerminate),
								CertificateRefs: []gatewayv1.SecretObjectReference{
									{
										Group: (*gatewayv1.Group)(ptr("")),
										Kind:  (*gatewayv1.Kind)(ptr("Secret")),
										Name:  gatewayv1.ObjectName(secret.Name),
									},
								},
							},
						},
					},
				},
			},
			check: func(t *testing.T, gw *gatewayv1.Gateway) {
				assert.Equal(t, "istio", string(gw.Spec.GatewayClassName))
				assert.Len(t, gw.Spec.Listeners, 1)
				assert.Equal(t, "https", string(gw.Spec.Listeners[0].Name))
				assert.Equal(t, int32(443), int32(gw.Spec.Listeners[0].Port))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create gateway
			gw, err := gatewayClientset.GatewayV1().Gateways(ns.Name).Create(
				context.Background(),
				tt.gateway,
				metav1.CreateOptions{},
			)
			require.NoError(t, err)

			// Wait for gateway to be ready
			err = waitForGateway(gatewayClientset, ns.Name, gw.Name)
			require.NoError(t, err)

			// Get gateway
			gw, err = gatewayClientset.GatewayV1().Gateways(ns.Name).Get(
				context.Background(),
				gw.Name,
				metav1.GetOptions{},
			)
			require.NoError(t, err)

			// Run checks
			tt.check(t, gw)
		})
	}
}

// waitForGateway waits for a gateway to be ready
func waitForGateway(client *gatewayclient.Clientset, namespace, name string) error {
	timeout := time.After(2 * time.Minute)
	tick := time.Tick(2 * time.Second)

	for {
		select {
		case <-timeout:
			// Get final status before giving up
			gw, err := client.GatewayV1().Gateways(namespace).Get(
				context.Background(),
				name,
				metav1.GetOptions{},
			)
			if err != nil {
				return fmt.Errorf("timeout and failed to get gateway: %v", err)
			}

			// Print all conditions for debugging
			var conditions []string
			for _, cond := range gw.Status.Conditions {
				conditions = append(conditions, fmt.Sprintf(
					"Type=%s Status=%s Reason=%s Message=%s",
					cond.Type, cond.Status, cond.Reason, cond.Message,
				))
			}
			return fmt.Errorf("timeout waiting for gateway %s/%s to be ready. Conditions: %s",
				namespace, name, strings.Join(conditions, "; "))

		case <-tick:
			gw, err := client.GatewayV1().Gateways(namespace).Get(
				context.Background(),
				name,
				metav1.GetOptions{},
			)
			if err != nil {
				fmt.Printf("Error getting gateway: %v\n", err)
				continue
			}

			// Print current conditions for debugging
			fmt.Printf("Gateway %s/%s conditions:\n", namespace, name)
			for _, cond := range gw.Status.Conditions {
				fmt.Printf("  %s: %s (Reason=%s, Message=%s)\n",
					cond.Type, cond.Status, cond.Reason, cond.Message)
			}

			// Check if gateway is ready
			for _, condition := range gw.Status.Conditions {
				if condition.Type == string(gatewayv1.GatewayConditionAccepted) &&
					condition.Status == metav1.ConditionTrue {
					return nil
				}
			}
		}
	}
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}

// randString generates a random string of the specified length
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
