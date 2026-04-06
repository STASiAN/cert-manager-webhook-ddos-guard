## Context

The repo is a cert-manager ACME DNS-01 webhook template. `customDNSProviderSolver` in `main.go` has three no-op methods: `Present`, `CleanUp`, and `Initialize`. This design replaces those stubs with a complete implementation against the DDoS-Guard DNS REST API (`https://webapi.ddos-guard.net`).

The DDoS-Guard API is a simple HTTP REST API using `POST application/x-www-form-urlencoded`. Every request carries `client_id` + `api_key` in the form body. Records are addressed by numeric IDs, not by name — so deleting a record requires a lookup first.

## Goals / Non-Goals

**Goals:**
- Implement `Present`: create the ACME TXT challenge record via `add-record`
- Implement `CleanUp`: find and delete only the matching TXT record via `list-records` + `delete-record`
- Implement `Initialize`: build a Kubernetes client for credential fetching
- Load `client_id` / `api_key` from a Kubernetes Secret referenced in per-issuer config
- Cache zone IDs to avoid redundant `list-dns` calls
- Handle FQDN trailing-dot normalisation (cert-manager uses `example.com.`, API expects `example.com`)

**Non-Goals:**
- Retry logic or circuit breaking (rely on cert-manager's built-in retry)
- Supporting DNS record types other than TXT
- Zone creation (zones must exist in DDoS-Guard before the solver is used)

## Decisions

### 1. Credential loading: per-request via Kubernetes client

**Decision**: Build a `kubernetes.Clientset` in `Initialize` and store it on the solver struct. `Present` and `CleanUp` fetch the Secret on every call.

**Rationale**: Credentials can be rotated without restarting the webhook. Caching them risks stale secrets. The extra k8s API call per challenge is negligible — challenges are infrequent.

**Alternative considered**: Read credentials once in `Initialize`. Rejected because cert-manager may reuse a running webhook for many issuers with different secrets.

---

### 2. Zone ID resolution: cached in-memory map

**Decision**: On first use of a zone, call `list-dns`, find the matching zone by domain name, and cache `domain → dns_id` in a `map[string]int` protected by `sync.RWMutex`.

**Rationale**: `dns_id` is stable for the lifetime of the zone. Calling `list-dns` on every `Present`/`CleanUp` is wasteful. Cache invalidation is not needed — if a zone is deleted, the solve fails and the operator must intervene anyway.

---

### 3. Zone matching: strip trailing dot, longest-suffix match

**Decision**: Strip the trailing dot from `ch.ResolvedZone` before comparing against DDoS-Guard zone `domain` values.

**Rationale**: cert-manager always appends a trailing dot to FQDNs and zone names. The DDoS-Guard API returns bare domain names (e.g., `example.com`).

**Longest-suffix match**: if both `example.com` and `sub.example.com` exist as separate zones, pick the more specific one. This matches standard DNS delegation behaviour.

---

### 4. Record name in add-record: strip trailing dot

**Decision**: Strip the trailing dot from `ch.ResolvedFQDN` when calling `add-record`.

**Rationale**: DDoS-Guard API returns and accepts bare hostnames. cert-manager provides FQDNs with trailing dots.

---

### 5. Single file vs. separate package

**Decision**: Implement everything in `main.go` (plus a small `ddosguard/client.go` package for the API calls).

**Rationale**: Keeping the API client in its own package makes it independently testable and keeps `main.go` focused on the cert-manager webhook interface. The package stays within this module (not published separately).

---

### 6. TTL for challenge records

**Decision**: Use TTL=120 (2 minutes) for TXT records created by `Present`.

**Rationale**: Low enough that stale records don't linger long after `CleanUp`, but high enough that DNS propagation is stable before cert-manager's self-check.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| `list-records` returns many records, making CleanUp slow | Acceptable — typical zones have <100 records; no pagination in the API |
| Zone cache is never invalidated | Solver restart clears it; cache misses are safe (just re-call `list-dns`) |
| Concurrent `Present` calls for the same FQDN could add duplicate TXT records | The DDoS-Guard API allows multiple TXT records with the same name; cert-manager's self-check and CleanUp handle this correctly |
| Secret read on every call adds latency | Negligible for challenge flow; no realistic SLA impact |

## Migration Plan

1. Rename Go module in `go.mod` to `github.com/stasian/cert-manager-webhook-ddos-guard`
2. Update all internal import paths
3. Implement `ddosguard` client package
4. Implement solver methods in `main.go`
5. Update Helm chart `values.yaml` (image, groupName)
6. Update `Dockerfile` and `README.md`
7. Wire real solver into `main_test.go` (keep example solver test for CI reference)

No rollback strategy needed — this is a new repository with no production deployments.

## Open Questions

- Should `groupName` default be set in `values.yaml` to `acme.stasian.dev` or left empty for users to configure? → Leave empty; document in README.
