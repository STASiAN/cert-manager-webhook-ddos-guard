package main

import (
	"os"
	"testing"

	acmetest "github.com/cert-manager/cert-manager/test/acme"

	"github.com/stasian/cert-manager-webhook-ddos-guard/example"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
)

func TestRunsSuite(t *testing.T) {
	// To run the full conformance suite against the real DDoS-Guard solver,
	// set TEST_ZONE_NAME and ensure testdata/my-custom-solver/config.json
	// contains valid clientIdSecretRef and apiKeySecretRef configuration.
	//
	// Example:
	//   TEST_ZONE_NAME=example.com. go test -v .
	//
	if zone != "" {
		fixture := acmetest.NewFixture(&customDNSProviderSolver{},
			acmetest.SetResolvedZone(zone),
			acmetest.SetAllowAmbientCredentials(false),
			acmetest.SetManifestPath("testdata/my-custom-solver"),
		)
		fixture.RunBasic(t)
		fixture.RunExtended(t)
		return
	}

	// Default: run against the in-memory example solver (no credentials needed)
	solver := example.New("59351")
	fixture := acmetest.NewFixture(solver,
		acmetest.SetResolvedZone("example.com."),
		acmetest.SetManifestPath("testdata/my-custom-solver"),
		acmetest.SetDNSServer("127.0.0.1:59351"),
		acmetest.SetUseAuthoritative(false),
	)
	fixture.RunBasic(t)
	fixture.RunExtended(t)
}
