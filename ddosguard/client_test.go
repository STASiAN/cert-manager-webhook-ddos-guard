package ddosguard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}
	return client, server
}

func TestListDNS(t *testing.T) {
	zones := []Zone{
		{ID: 1, Domain: "example.com"},
		{ID: 2, Domain: "test.org"},
	}
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "list-dns" {
			t.Errorf("expected action=list-dns, got %s", r.URL.Query().Get("action"))
		}
		if r.FormValue("client_id") != "123" {
			t.Errorf("expected client_id=123, got %s", r.FormValue("client_id"))
		}
		if r.FormValue("api_key") != "secret" {
			t.Errorf("expected api_key=secret, got %s", r.FormValue("api_key"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zones)
	})
	defer server.Close()

	result, err := client.ListDNS("123", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(result))
	}
	if result[0].Domain != "example.com" {
		t.Errorf("expected example.com, got %s", result[0].Domain)
	}
}

func TestListRecords(t *testing.T) {
	records := []Record{
		{ID: 10, Name: "_acme-challenge.example.com", Type: "TXT", Content: "token123", TTL: 120},
	}
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "list-records" {
			t.Errorf("expected action=list-records, got %s", r.URL.Query().Get("action"))
		}
		if r.FormValue("dns_id") != "1" {
			t.Errorf("expected dns_id=1, got %s", r.FormValue("dns_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	})
	defer server.Close()

	result, err := client.ListRecords("123", "secret", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}
	if result[0].Content != "token123" {
		t.Errorf("expected token123, got %s", result[0].Content)
	}
}

func TestAddRecord(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "add-record" {
			t.Errorf("expected action=add-record, got %s", r.URL.Query().Get("action"))
		}
		if r.FormValue("name") != "_acme-challenge.example.com" {
			t.Errorf("unexpected name: %s", r.FormValue("name"))
		}
		if r.FormValue("type") != "TXT" {
			t.Errorf("unexpected type: %s", r.FormValue("type"))
		}
		if r.FormValue("content") != "mytoken" {
			t.Errorf("unexpected content: %s", r.FormValue("content"))
		}
		if r.FormValue("ttl") != "120" {
			t.Errorf("unexpected ttl: %s", r.FormValue("ttl"))
		}
		record := Record{ID: 42, Name: r.FormValue("name"), Type: "TXT", Content: "mytoken", TTL: 120}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(record)
	})
	defer server.Close()

	result, err := client.AddRecord("123", "secret", 1, "_acme-challenge.example.com", "TXT", "mytoken", 120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != 42 {
		t.Errorf("expected ID 42, got %d", result.ID)
	}
}

func TestDeleteRecord(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "delete-record" {
			t.Errorf("expected action=delete-record, got %s", r.URL.Query().Get("action"))
		}
		if r.FormValue("record_id") != "42" {
			t.Errorf("expected record_id=42, got %s", r.FormValue("record_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[]}`))
	})
	defer server.Close()

	err := client.DeleteRecord("123", "secret", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIError(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(apiError{Message: "Authentication error", Status: 403})
	})
	defer server.Close()

	_, err := client.ListDNS("bad", "creds")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "ddos-guard API error (HTTP 403): Authentication error" {
		t.Errorf("unexpected error message: %s", got)
	}
}
