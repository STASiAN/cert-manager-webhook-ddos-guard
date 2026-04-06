package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"github.com/stasian/cert-manager-webhook-ddos-guard/ddosguard"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&customDNSProviderSolver{},
	)
}

type customDNSProviderSolver struct {
	client    kubernetes.Interface
	dnsClient *ddosguard.Client
	zoneCache map[string]int
	zoneMu    sync.RWMutex
}

type customDNSProviderConfig struct {
	ClientIdSecretRef cmmeta.SecretKeySelector `json:"clientIdSecretRef"`
	ApiKeySecretRef   cmmeta.SecretKeySelector `json:"apiKeySecretRef"`
}

func (c *customDNSProviderSolver) Name() string {
	return "ddos-guard"
}

func (c *customDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	clientID, apiKey, err := c.loadCredentials(cfg, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	zone := stripTrailingDot(ch.ResolvedZone)
	dnsID, err := c.resolveZoneID(clientID, apiKey, zone)
	if err != nil {
		return err
	}

	fqdn := stripTrailingDot(ch.ResolvedFQDN)
	_, err = c.dnsClient.AddRecord(clientID, apiKey, dnsID, fqdn, "TXT", ch.Key, 120)
	return err
}

func (c *customDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	clientID, apiKey, err := c.loadCredentials(cfg, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	zone := stripTrailingDot(ch.ResolvedZone)
	dnsID, err := c.resolveZoneID(clientID, apiKey, zone)
	if err != nil {
		return err
	}

	records, err := c.dnsClient.ListRecords(clientID, apiKey, dnsID)
	if err != nil {
		return err
	}

	fqdn := stripTrailingDot(ch.ResolvedFQDN)
	for _, r := range records {
		if r.Type == "TXT" && r.Name == fqdn && r.Content == ch.Key {
			return c.dnsClient.DeleteRecord(clientID, apiKey, r.ID)
		}
	}

	// Record not found — already cleaned up
	return nil
}

func (c *customDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	c.client = cl
	c.dnsClient = ddosguard.NewClient()
	c.zoneCache = make(map[string]int)
	return nil
}

func (c *customDNSProviderSolver) loadCredentials(cfg customDNSProviderConfig, namespace string) (string, string, error) {
	clientIDSecret, err := c.client.CoreV1().Secrets(namespace).Get(context.TODO(), cfg.ClientIdSecretRef.LocalObjectReference.Name, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get client_id secret %q: %w", cfg.ClientIdSecretRef.LocalObjectReference.Name, err)
	}
	clientIDBytes, ok := clientIDSecret.Data[cfg.ClientIdSecretRef.Key]
	if !ok {
		return "", "", fmt.Errorf("key %q not found in secret %q", cfg.ClientIdSecretRef.Key, cfg.ClientIdSecretRef.LocalObjectReference.Name)
	}

	apiKeySecret, err := c.client.CoreV1().Secrets(namespace).Get(context.TODO(), cfg.ApiKeySecretRef.LocalObjectReference.Name, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get api_key secret %q: %w", cfg.ApiKeySecretRef.LocalObjectReference.Name, err)
	}
	apiKeyBytes, ok := apiKeySecret.Data[cfg.ApiKeySecretRef.Key]
	if !ok {
		return "", "", fmt.Errorf("key %q not found in secret %q", cfg.ApiKeySecretRef.Key, cfg.ApiKeySecretRef.LocalObjectReference.Name)
	}

	return strings.TrimSpace(string(clientIDBytes)), strings.TrimSpace(string(apiKeyBytes)), nil
}

func (c *customDNSProviderSolver) resolveZoneID(clientID, apiKey, zone string) (int, error) {
	c.zoneMu.RLock()
	if id, ok := c.zoneCache[zone]; ok {
		c.zoneMu.RUnlock()
		return id, nil
	}
	c.zoneMu.RUnlock()

	zones, err := c.dnsClient.ListDNS(clientID, apiKey)
	if err != nil {
		return 0, err
	}

	// Find the longest matching zone (most specific)
	var bestMatch ddosguard.Zone
	for _, z := range zones {
		if z.Domain == zone || strings.HasSuffix(zone, "."+z.Domain) {
			if len(z.Domain) > len(bestMatch.Domain) {
				bestMatch = z
			}
		}
	}

	if bestMatch.Domain == "" {
		return 0, fmt.Errorf("zone %q not found in DDoS-Guard DNS zones", zone)
	}

	c.zoneMu.Lock()
	c.zoneCache[zone] = bestMatch.ID
	c.zoneMu.Unlock()

	return bestMatch.ID, nil
}

func stripTrailingDot(s string) string {
	return strings.TrimSuffix(s, ".")
}

func loadConfig(cfgJSON *extapi.JSON) (customDNSProviderConfig, error) {
	cfg := customDNSProviderConfig{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}
	return cfg, nil
}
