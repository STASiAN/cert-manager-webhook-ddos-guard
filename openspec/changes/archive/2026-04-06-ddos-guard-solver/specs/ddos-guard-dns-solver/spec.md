## ADDED Requirements

### Requirement: Present creates a TXT record
The solver SHALL create a DNS TXT record via the DDoS-Guard API when `Present` is called. The record name SHALL be `ch.ResolvedFQDN` (with trailing dot stripped) and the record content SHALL be `ch.Key`. The record type SHALL be `TXT` with a TTL of 120 seconds.

#### Scenario: Successful TXT record creation
- **WHEN** cert-manager calls `Present` with a valid `ChallengeRequest`
- **THEN** the solver calls `add-record` on the DDoS-Guard API with `dns_id` matching the zone, `name` set to the resolved FQDN (trailing dot stripped), `type=TXT`, `content` set to the challenge key, and `ttl=120`

#### Scenario: Present is called multiple times with the same value
- **WHEN** cert-manager calls `Present` twice with the same FQDN and key
- **THEN** the solver SHALL tolerate the duplicate call without returning an error (a second TXT record may be created; this is acceptable)

#### Scenario: Present fails due to API error
- **WHEN** the DDoS-Guard API returns an HTTP error or non-200 status
- **THEN** the solver SHALL return an error wrapping the API response details

---

### Requirement: CleanUp deletes only the matching TXT record
The solver SHALL delete only the TXT record whose name matches `ch.ResolvedFQDN` (trailing dot stripped) and whose content matches `ch.Key`. Other TXT records with the same name but different content MUST NOT be deleted.

#### Scenario: Successful single-record cleanup
- **WHEN** cert-manager calls `CleanUp` with a valid `ChallengeRequest`
- **THEN** the solver calls `list-records` for the zone, finds the record matching both the FQDN and the key, and calls `delete-record` with that record's numeric ID

#### Scenario: Multiple TXT records exist for the same FQDN
- **WHEN** multiple TXT records exist for the same FQDN (e.g., concurrent wildcard + base domain challenges)
- **THEN** the solver SHALL delete only the record whose content matches `ch.Key`, leaving other records intact

#### Scenario: No matching record found during cleanup
- **WHEN** `list-records` returns no record matching both the FQDN and the key
- **THEN** the solver SHALL return nil (no error) since the record is already gone

#### Scenario: CleanUp fails due to API error
- **WHEN** the DDoS-Guard API returns an error during `list-records` or `delete-record`
- **THEN** the solver SHALL return an error wrapping the API response details

---

### Requirement: Initialize builds a Kubernetes client
The solver SHALL create a `kubernetes.Clientset` from the provided `kubeClientConfig` during `Initialize` and store it on the solver struct for later use in `Present` and `CleanUp`.

#### Scenario: Successful initialization
- **WHEN** the webhook starts and `Initialize` is called with a valid `kubeClientConfig`
- **THEN** a Kubernetes clientset is created and stored on the solver struct, and nil is returned

#### Scenario: Invalid kubeClientConfig
- **WHEN** `Initialize` is called with a config that fails to create a client
- **THEN** the solver SHALL return the underlying error

---

### Requirement: Credentials loaded from Kubernetes Secret
The solver SHALL read `client_id` and `api_key` from a Kubernetes Secret referenced in the per-issuer `customDNSProviderConfig`. The config SHALL contain `apiKeySecretRef` with fields `name` and `key`, and a `clientIdSecretRef` with fields `name` and `key`. Both Secret references MUST resolve in the same namespace as the solver deployment.

#### Scenario: Successful credential loading
- **WHEN** `Present` or `CleanUp` is called and the referenced Secret exists
- **THEN** the solver reads `client_id` and `api_key` from the Secret data and uses them for DDoS-Guard API authentication

#### Scenario: Secret does not exist
- **WHEN** the referenced Secret does not exist or the key is missing
- **THEN** the solver SHALL return an error describing which Secret/key could not be found

---

### Requirement: Zone ID resolution with caching
The solver SHALL resolve the DDoS-Guard numeric `dns_id` for a given zone by calling `list-dns` and matching the zone domain (with trailing dot stripped). Resolved zone IDs SHALL be cached in memory for the lifetime of the process.

#### Scenario: Zone ID resolved on first call
- **WHEN** `Present` or `CleanUp` is called for a zone not yet in cache
- **THEN** the solver calls `list-dns`, finds the zone whose `domain` matches the resolved zone (trailing dot stripped), caches the mapping, and uses the `dns_id`

#### Scenario: Zone ID served from cache on subsequent calls
- **WHEN** `Present` or `CleanUp` is called for a zone already resolved
- **THEN** the solver uses the cached `dns_id` without calling `list-dns`

#### Scenario: Zone not found in DDoS-Guard
- **WHEN** `list-dns` returns no zone matching the resolved zone domain
- **THEN** the solver SHALL return an error indicating the zone was not found

#### Scenario: Longest-suffix zone matching
- **WHEN** `list-dns` returns multiple zones where more than one is a suffix of the FQDN (e.g., `example.com` and `sub.example.com`)
- **THEN** the solver SHALL select the most specific (longest) matching zone

---

### Requirement: FQDN trailing-dot normalization
The solver SHALL strip trailing dots from `ch.ResolvedFQDN` and `ch.ResolvedZone` before using them in DDoS-Guard API calls, since the DDoS-Guard API uses bare domain names without trailing dots.

#### Scenario: FQDN with trailing dot
- **WHEN** cert-manager provides `_acme-challenge.example.com.` as the resolved FQDN
- **THEN** the solver uses `_acme-challenge.example.com` (no trailing dot) in the API call

#### Scenario: Zone with trailing dot
- **WHEN** cert-manager provides `example.com.` as the resolved zone
- **THEN** the solver matches against the DDoS-Guard zone `example.com` (no trailing dot)
