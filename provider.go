package immosquare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Version of the immosquare DNS provider.
const Version = "1.1.0"

// defaultMinTTL is the minimum TTL applied to records created via this provider.
// Prevents issues with TTL 0 (e.g. certmagic ACME challenges) falling back to
// high zone defaults like 1800s, which slows down DNS propagation.
const defaultMinTTL = 120 * time.Second

// Provider manages the DNS records of the zones served by immosquare through
// the immosquare DNS API. It implements the libdns interfaces and is
// registered as the Caddy module dns.providers.immosquare (see module.go).
type Provider struct {
	APIToken string `json:"api_token,omitempty"`
	Endpoint string `json:"endpoint"`
	client   *http.Client
}

// apiRecord is a record as listed by GET /zones/{zone}/records. Names are
// fully qualified without a trailing dot; an MX carries its preference and
// target in dedicated fields, every other type carries its value in data.
type apiRecord struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	TTL        int    `json:"ttl"`
	Data       string `json:"data"`
	Preference uint16 `json:"preference"`
	Target     string `json:"target"`
}

// initClient initializes the HTTP client if necessary
func (p *Provider) initClient() error {
	if p.client == nil {
		p.client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	if p.Endpoint == "" {
		return fmt.Errorf("endpoint is required for the immosquare provider")
	}
	return nil
}

// makeRequest makes an HTTP request to the immosquare DNS API
func (p *Provider) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	if err := p.initClient(); err != nil {
		return nil, err
	}

	url := p.Endpoint + path
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("JSON serialization error: %w", err)
		}
		req, err = http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(jsonBody)))
		if err != nil {
			return nil, fmt.Errorf("request creation error: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("request creation error: %w", err)
		}
	}

	if p.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIToken)
	}

	return p.client.Do(req)
}

// checkResponse turns any non-2xx answer into an error. Callers such as
// certmagic only learn from the returned error that a record was not
// written or not deleted: a swallowed failure leaves stale _acme-challenge
// records behind until the zone hits its TXT limit and renewals fail.
func checkResponse(resp *http.Response, action string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("API error during %s: %s %s", action, resp.Status, strings.TrimSpace(string(body)))
}

// sendRecords posts records to the API with the given method. A TTL below
// minTTL is raised to it; pass 0 to send the TTLs unchanged.
func (p *Provider) sendRecords(ctx context.Context, method, zone, action string, records []libdns.Record, minTTL time.Duration) ([]libdns.Record, error) {
	if len(records) == 0 {
		return []libdns.Record{}, nil
	}

	apiRecords := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		rr := record.RR()
		ttl := rr.TTL
		if ttl < minTTL {
			ttl = minTTL
		}
		apiRecords = append(apiRecords, map[string]interface{}{
			"name": rr.Name,
			"type": rr.Type,
			"data": rr.Data,
			"ttl":  int(ttl.Seconds()),
		})
	}

	resp, err := p.makeRequest(ctx, method, "/zones/"+zone+"/records", map[string]interface{}{"records": apiRecords})
	if err != nil {
		return nil, fmt.Errorf("%s request error: %w", method, err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp, action); err != nil {
		return nil, err
	}

	return parseRecords(records), nil
}

// parseRecord returns the typed libdns record of rr, or rr itself when its
// data cannot be parsed (unknown type, malformed value).
func parseRecord(rr libdns.RR) libdns.Record {
	record, err := rr.Parse()
	if err != nil {
		return rr
	}
	return record
}

// parseRecords converts records to their typed libdns structures.
func parseRecords(records []libdns.Record) []libdns.Record {
	result := make([]libdns.Record, 0, len(records))
	for _, record := range records {
		result = append(result, parseRecord(record.RR()))
	}
	return result
}

// GetRecords lists all DNS records of the zone, with names relative to it.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	resp, err := p.makeRequest(ctx, "GET", "/zones/"+zone+"/records", nil)
	if err != nil {
		return nil, fmt.Errorf("GET request error: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp, "listing"); err != nil {
		return nil, err
	}

	var payload struct {
		Records []apiRecord `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("JSON decoding error: %w", err)
	}

	records := make([]libdns.Record, 0, len(payload.Records))
	for _, record := range payload.Records {
		entryType := strings.ToUpper(record.Type)
		data := record.Data
		if entryType == "MX" {
			data = fmt.Sprintf("%d %s", record.Preference, record.Target)
		}
		records = append(records, parseRecord(libdns.RR{
			Name: libdns.RelativeName(record.Name, zone),
			Type: entryType,
			Data: data,
			TTL:  time.Duration(record.TTL) * time.Second,
		}))
	}

	return records, nil
}

// AppendRecords adds new DNS records to the zone without touching the
// existing ones. Returns the records that have been added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.sendRecords(ctx, "POST", zone, "addition", records, defaultMinTTL)
}

// SetRecords makes the given records the only members of their RRset
// (name, type) in the zone; other records are left untouched. The API
// applies it atomically. Returns the records that have been set.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.sendRecords(ctx, "PUT", zone, "update", records, defaultMinTTL)
}

// DeleteRecords deletes the specified DNS records from the zone.
// Returns the records that have been deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.sendRecords(ctx, "DELETE", zone, "deletion", records, 0)
}

// Interface guards to ensure the Provider implements all libdns interfaces
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
