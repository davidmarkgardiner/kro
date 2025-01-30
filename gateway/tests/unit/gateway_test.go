package unit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// TestGatewayConfig_Validation tests the validation of GatewayConfig fields
func TestGatewayConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		gateway *GatewayConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			gateway: &GatewayConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gateway",
					Namespace: "default",
				},
				Spec: GatewayConfigSpec{
					Name:               "test-gateway",
					Hostname:           "example.com",
					TLSSecretName:      "tls-secret",
					TLSSecretNamespace: "cert-manager",
					Port:               443,
					Protocol:           "HTTPS",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			gateway: &GatewayConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: GatewayConfigSpec{
					Hostname:           "example.com",
					TLSSecretName:      "tls-secret",
					TLSSecretNamespace: "cert-manager",
				},
			},
			wantErr: true,
		},
		{
			name: "missing hostname",
			gateway: &GatewayConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gateway",
					Namespace: "default",
				},
				Spec: GatewayConfigSpec{
					Name:               "test-gateway",
					TLSSecretName:      "tls-secret",
					TLSSecretNamespace: "cert-manager",
				},
			},
			wantErr: true,
		},
		{
			name: "missing TLS secret name",
			gateway: &GatewayConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gateway",
					Namespace: "default",
				},
				Spec: GatewayConfigSpec{
					Name:               "test-gateway",
					Hostname:           "example.com",
					TLSSecretNamespace: "cert-manager",
				},
			},
			wantErr: true,
		},
		{
			name: "missing TLS secret namespace",
			gateway: &GatewayConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gateway",
					Namespace: "default",
				},
				Spec: GatewayConfigSpec{
					Name:          "test-gateway",
					Hostname:      "example.com",
					TLSSecretName: "tls-secret",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGatewayConfig(tt.gateway)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// validateGatewayConfig validates a GatewayConfig resource
func validateGatewayConfig(gateway *GatewayConfig) error {
	if gateway.Name == "" {
		return fmt.Errorf("name is required")
	}
	if gateway.Spec.Name == "" {
		return fmt.Errorf("spec.name is required")
	}
	if gateway.Spec.Hostname == "" {
		return fmt.Errorf("spec.hostname is required")
	}
	if gateway.Spec.TLSSecretName == "" {
		return fmt.Errorf("spec.tlsSecretName is required")
	}
	if gateway.Spec.TLSSecretNamespace == "" {
		return fmt.Errorf("spec.tlsSecretNamespace is required")
	}
	return nil
}

// TestResource_Generation tests the generation of Kubernetes Gateway resources
func TestResource_Generation(t *testing.T) {
	gateway := &GatewayConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gateway",
			Namespace: "default",
		},
		Spec: GatewayConfigSpec{
			Name:               "test-gateway",
			Hostname:           "example.com",
			Namespace:          "aks-istio-ingress",
			TLSSecretName:      "tls-secret",
			TLSSecretNamespace: "cert-manager",
			Port:               443,
			Protocol:           "HTTPS",
		},
	}

	t.Run("Gateway", func(t *testing.T) {
		gw := generateGateway(gateway)
		require.NotNil(t, gw)
		assert.Equal(t, gateway.Spec.Name, gw.Name)
		assert.Equal(t, gateway.Spec.Namespace, gw.Namespace)

		// Verify gateway class
		assert.Equal(t, "istio", string(gw.Spec.GatewayClassName))

		// Verify listeners
		require.Len(t, gw.Spec.Listeners, 1)
		listener := gw.Spec.Listeners[0]
		assert.Equal(t, "private-zone", string(listener.Name))
		assert.Equal(t, gateway.Spec.Hostname, string(*listener.Hostname))
		assert.Equal(t, gateway.Spec.Protocol, string(listener.Protocol))
		assert.Equal(t, gatewayv1.PortNumber(gateway.Spec.Port), listener.Port)

		// Verify TLS configuration
		require.NotNil(t, listener.TLS)
		assert.Equal(t, ptr(gatewayv1.TLSModeTerminate), listener.TLS.Mode)
		require.Len(t, listener.TLS.CertificateRefs, 1)
		certRef := listener.TLS.CertificateRefs[0]
		assert.Equal(t, gateway.Spec.TLSSecretName, string(certRef.Name))
		assert.Equal(t, gateway.Spec.TLSSecretNamespace, string(*certRef.Namespace))
	})
}

// generateGateway generates a Gateway resource from a GatewayConfig
func generateGateway(config *GatewayConfig) *gatewayv1.Gateway {
	hostname := gatewayv1.Hostname(config.Spec.Hostname)
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Spec.Name,
			Namespace: config.Spec.Namespace,
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "istio",
			Listeners: []gatewayv1.Listener{
				{
					Name:     gatewayv1.SectionName("private-zone"),
					Hostname: &hostname,
					Port:     gatewayv1.PortNumber(config.Spec.Port),
					Protocol: gatewayv1.ProtocolType(config.Spec.Protocol),
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: ptr(gatewayv1.NamespacesFromAll),
						},
					},
					TLS: &gatewayv1.GatewayTLSConfig{
						Mode: ptr(gatewayv1.TLSModeTerminate),
						CertificateRefs: []gatewayv1.SecretObjectReference{
							{
								Group: (*gatewayv1.Group)(ptr("")),
								Kind:  (*gatewayv1.Kind)(ptr("Secret")),
								Name:  gatewayv1.ObjectName(config.Spec.TLSSecretName),
								Namespace: (*gatewayv1.Namespace)(ptr(gatewayv1.Namespace(
									config.Spec.TLSSecretNamespace,
								))),
							},
						},
					},
				},
			},
		},
	}
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}
