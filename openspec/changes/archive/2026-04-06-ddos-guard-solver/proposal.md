## Why

This repo ships as a template with no-op `Present` and `CleanUp` methods. Users who run cert-manager with DDoS-Guard DNS hosting must implement the solver themselves. This change delivers a complete, working solver so the repo can be used directly rather than treated as boilerplate.

## What Changes

- **BREAKING**: Module path renamed from `github.com/cert-manager/webhook-example` to `github.com/stasian/cert-manager-webhook-ddos-guard`
- Replace `customDNSProviderSolver` stub in `main.go` with a full DDoS-Guard DNS API implementation
- Implement `Present`: call `add-record` to create a TXT record for the ACME challenge
- Implement `CleanUp`: call `list-records` then `delete-record` to remove only the matching TXT record
- Implement `Initialize`: build an authenticated HTTP client from credentials stored in a Kubernetes Secret
- Add `customDNSProviderConfig` fields: `clientId` and `apiKeySecretRef`
- Update Helm chart values and deployment to reference the new image and group name
- Update `main_test.go` to run the conformance suite against the real solver (integration test)
- Update `go.mod`, `README.md`, and `Dockerfile` for the new module/image

## Capabilities

### New Capabilities
- `ddos-guard-dns-solver`: Full cert-manager ACME DNS-01 solver backed by the DDoS-Guard REST API (`https://webapi.ddos-guard.net`). Covers zone ID resolution, TXT record creation, targeted TXT record deletion, credential loading from a Kubernetes Secret, and zone ID caching.

### Modified Capabilities

_(none — no existing specs)_

## Impact

- `main.go`: all three solver methods fully implemented; config struct updated
- `go.mod`: module path changed (**BREAKING**)
- `deploy/example-webhook/`: Helm chart values updated (image, groupName)
- `main_test.go`: conformance test wired to real solver
- New dependency: standard library `net/http` only (no new external deps needed beyond what's already imported)
- Kubernetes RBAC: solver ServiceAccount needs `get` on Secrets in its own namespace
