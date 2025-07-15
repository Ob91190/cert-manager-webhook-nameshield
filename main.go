package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&nameshieldDNSSolver{},
	)
}

// nameshieldDNSSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for NameShield DNS provider.
// To do so, it must implement the `github.com/cert-manager/cert-manager/pkg/acme/webhook.Solver`
// interface.
type nameshieldDNSSolver struct {
	client kubernetes.Interface
}

// nameshieldDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
type nameshieldDNSProviderConfig struct {
	APIKeySecretRef v1alpha1.SecretKeySelector `json:"apiKeySecretRef"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
func (c *nameshieldDNSSolver) Name() string {
	return "nameshield"
}

// Present is responsible for actually presenting the DNS record with the
// NameShield DNS provider.
func (c *nameshieldDNSSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("error decoding solver config: %v", err)
	}

	// Get the API key from the referenced secret
	apiKey, err := c.getAPIKey(cfg, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("error getting API key: %v", err)
	}

	// Create NameShield client
	client := NewNameShieldClient(apiKey)

	// Extract domain from the FQDN (remove the challenge prefix)
	domain := c.extractDomain(ch.ResolvedFQDN)
	
	// Create the TXT record
	recordName := c.extractRecordName(ch.ResolvedFQDN, domain)
	
	return client.CreateTxtRecord(&domain, &recordName, &ch.Key, 300)
}

// CleanUp should delete the relevant TXT record from the NameShield DNS provider.
func (c *nameshieldDNSSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("error decoding solver config: %v", err)
	}

	// Get the API key from the referenced secret
	apiKey, err := c.getAPIKey(cfg, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("error getting API key: %v", err)
	}

	// Create NameShield client
	client := NewNameShieldClient(apiKey)

	// Extract domain from the FQDN
	domain := c.extractDomain(ch.ResolvedFQDN)
	
	// Delete the TXT record
	recordName := c.extractRecordName(ch.ResolvedFQDN, domain)
	
	return client.DeleteTxtRecord(&domain, &recordName)
}

// Initialize will be called when the webhook first starts.
func (c *nameshieldDNSSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	c.client = cl
	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (nameshieldDNSProviderConfig, error) {
	cfg := nameshieldDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

// getAPIKey retrieves the API key from the referenced Kubernetes secret
func (c *nameshieldDNSSolver) getAPIKey(cfg nameshieldDNSProviderConfig, namespace string) (string, error) {
	secretName := cfg.APIKeySecretRef.Name
	secretKey := cfg.APIKeySecretRef.Key

	secret, err := c.client.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting secret %s/%s: %v", namespace, secretName, err)
	}

	if data, ok := secret.Data[secretKey]; ok {
		return string(data), nil
	}
	return "", fmt.Errorf("no key %s in secret %s/%s", secretKey, namespace, secretName)
}

// extractDomain extracts the domain from the FQDN
// Example: _acme-challenge.example.com. -> example.com
func (c *nameshieldDNSSolver) extractDomain(fqdn string) string {
	// Remove trailing dot if present
	if strings.HasSuffix(fqdn, ".") {
		fqdn = fqdn[:len(fqdn)-1]
	}
	
	// Remove _acme-challenge prefix
	if strings.HasPrefix(fqdn, "_acme-challenge.") {
		return fqdn[16:] // len("_acme-challenge.") = 16
	}
	
	return fqdn
}

// extractRecordName extracts the record name from FQDN
// Example: _acme-challenge.sub.example.com, example.com -> _acme-challenge.sub
func (c *nameshieldDNSSolver) extractRecordName(fqdn, domain string) string {
	// Remove trailing dot if present
	if strings.HasSuffix(fqdn, ".") {
		fqdn = fqdn[:len(fqdn)-1]
	}
	
	// If FQDN is just _acme-challenge.domain, return _acme-challenge
	if fqdn == "_acme-challenge."+domain {
		return "_acme-challenge"
	}
	
	// Remove domain suffix to get the record name
	if strings.HasSuffix(fqdn, "."+domain) {
		return fqdn[:len(fqdn)-len(domain)-1]
	}
	
	return fqdn
}
