package ddosguard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const DefaultBaseURL = "https://webapi.ddos-guard.net"

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		HTTPClient: http.DefaultClient,
	}
}

type Zone struct {
	ID     int    `json:"id"`
	Domain string `json:"domain"`
}

type Record struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

type apiError struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (c *Client) doRequest(action string, params url.Values) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/api-dns?action=%s", c.BaseURL, action)

	resp, err := c.HTTPClient.PostForm(reqURL, params)
	if err != nil {
		return nil, fmt.Errorf("ddos-guard API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ddos-guard API: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("ddos-guard API error (HTTP %d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("ddos-guard API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func authParams(clientID, apiKey string) url.Values {
	return url.Values{
		"client_id": {clientID},
		"api_key":   {apiKey},
	}
}

func (c *Client) ListDNS(clientID, apiKey string) ([]Zone, error) {
	body, err := c.doRequest("list-dns", authParams(clientID, apiKey))
	if err != nil {
		return nil, err
	}

	var zones []Zone
	if err := json.Unmarshal(body, &zones); err != nil {
		return nil, fmt.Errorf("ddos-guard API: failed to decode list-dns response: %w", err)
	}
	return zones, nil
}

func (c *Client) ListRecords(clientID, apiKey string, dnsID int) ([]Record, error) {
	params := authParams(clientID, apiKey)
	params.Set("dns_id", strconv.Itoa(dnsID))

	body, err := c.doRequest("list-records", params)
	if err != nil {
		return nil, err
	}

	var records []Record
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("ddos-guard API: failed to decode list-records response: %w", err)
	}
	return records, nil
}

func (c *Client) AddRecord(clientID, apiKey string, dnsID int, name, recordType, content string, ttl int) (*Record, error) {
	params := authParams(clientID, apiKey)
	params.Set("dns_id", strconv.Itoa(dnsID))
	params.Set("name", name)
	params.Set("type", recordType)
	params.Set("content", content)
	params.Set("ttl", strconv.Itoa(ttl))

	body, err := c.doRequest("add-record", params)
	if err != nil {
		return nil, err
	}

	var record Record
	if err := json.Unmarshal(body, &record); err != nil {
		return nil, fmt.Errorf("ddos-guard API: failed to decode add-record response: %w", err)
	}
	return &record, nil
}

func (c *Client) DeleteRecord(clientID, apiKey string, recordID int) error {
	params := authParams(clientID, apiKey)
	params.Set("record_id", strconv.Itoa(recordID))

	_, err := c.doRequest("delete-record", params)
	return err
}
