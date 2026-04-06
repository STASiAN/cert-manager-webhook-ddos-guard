## 1. Module & Project Setup

- [x] 1.1 Rename module path in `go.mod` from `github.com/cert-manager/webhook-example` to `github.com/stasian/cert-manager-webhook-ddos-guard`
- [x] 1.2 Update all import paths in `main.go`, `main_test.go`, and `example/` to use the new module path
- [x] 1.3 Run `go mod tidy` and verify the build compiles

## 2. DDoS-Guard API Client Package

- [x] 2.1 Create `ddosguard/client.go` with `Client` struct holding base URL and HTTP client
- [x] 2.2 Implement `ListDNS(clientId, apiKey)` — calls `?action=list-dns`, returns `[]Zone{ID, Domain}`
- [x] 2.3 Implement `ListRecords(clientId, apiKey, dnsId)` — calls `?action=list-records`, returns `[]Record{ID, Name, Type, Content, TTL}`
- [x] 2.4 Implement `AddRecord(clientId, apiKey, dnsId, name, recordType, content, ttl)` — calls `?action=add-record`
- [x] 2.5 Implement `DeleteRecord(clientId, apiKey, recordId)` — calls `?action=delete-record`
- [x] 2.6 Add error handling: wrap non-200 responses with status and body in returned error

## 3. Solver Core Implementation

- [x] 3.1 Update `customDNSProviderSolver` struct: add `kubernetes.Clientset`, zone cache `map[string]int`, and `sync.RWMutex`
- [x] 3.2 Update `customDNSProviderConfig` struct: add `ClientIdSecretRef` and `ApiKeySecretRef` (type `cmmeta.SecretKeySelector`)
- [x] 3.3 Implement `Initialize`: create `kubernetes.Clientset` from `kubeClientConfig`, store on struct
- [x] 3.4 Implement credential loading helper: read `client_id` and `api_key` from the referenced Kubernetes Secrets
- [x] 3.5 Implement zone ID resolution helper: strip trailing dot, call `ListDNS`, longest-suffix match, cache result
- [x] 3.6 Implement `Present`: load creds → resolve zone ID → strip trailing dot from FQDN → call `AddRecord` with type=TXT, ttl=120
- [x] 3.7 Implement `CleanUp`: load creds → resolve zone ID → call `ListRecords` → find record matching FQDN+key → call `DeleteRecord`; return nil if no match found

## 4. Helm Chart & Deployment

- [x] 4.1 Update `deploy/example-webhook/values.yaml`: image repo, groupName placeholder
- [x] 4.2 Update `deploy/example-webhook/templates/deployment.yaml` if image references changed
- [x] 4.3 Update RBAC in `deploy/example-webhook/templates/rbac.yaml`: add `get` on Secrets

## 5. Dockerfile & README

- [x] 5.1 Update `Dockerfile`: change module path in build stage
- [x] 5.2 Update `README.md`: document DDoS-Guard solver, config fields, Secret setup, and Helm install instructions

## 6. Tests

- [x] 6.1 Add unit tests for `ddosguard/client.go` using `httptest.NewServer` to mock API responses
- [x] 6.2 Update `main_test.go`: uncomment and wire the real `customDNSProviderSolver` fixture (with env-based credentials for integration test)
- [x] 6.3 Verify `go test ./...` passes
