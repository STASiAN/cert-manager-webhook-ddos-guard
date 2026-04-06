<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# cert-manager-webhook-ddos-guard

A [cert-manager](https://cert-manager.io/) ACME DNS-01 webhook solver for
[DDoS-Guard](https://ddos-guard.net/) DNS hosting.

## Prerequisites

- A Kubernetes cluster with cert-manager installed
- A DDoS-Guard account with DNS hosting and API access
- Your domain's DNS zone must already exist in DDoS-Guard

## Installation

### 1. Create a Kubernetes Secret with your DDoS-Guard credentials

```bash
kubectl create secret generic ddos-guard-credentials \
  --namespace=cert-manager \
  --from-literal=client-id='YOUR_CLIENT_ID' \
  --from-literal=api-key='YOUR_API_KEY'
```

### 2. Install the webhook with Helm

```bash
helm install cert-manager-webhook-ddos-guard \
  --namespace=cert-manager \
  --set groupName='acme.stasian.dev' \
  deploy/example-webhook
```

The `groupName` must be a unique domain name you own. It is used as the
Kubernetes API group for the webhook.

### 3. Configure an Issuer

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
      - dns01:
          webhook:
            groupName: acme.stasian.dev
            solverName: ddos-guard
            config:
              clientIdSecretRef:
                name: ddos-guard-credentials
                key: client-id
              apiKeySecretRef:
                name: ddos-guard-credentials
                key: api-key
```

## Configuration

The webhook config accepts:

| Field | Type | Description |
|-------|------|-------------|
| `clientIdSecretRef.name` | string | Name of the Secret containing the DDoS-Guard client ID |
| `clientIdSecretRef.key` | string | Key within the Secret |
| `apiKeySecretRef.name` | string | Name of the Secret containing the DDoS-Guard API key |
| `apiKeySecretRef.key` | string | Key within the Secret |

## Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite.

```bash
TEST_ZONE_NAME=example.com. make test
```
