package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Deliverer submits mapped records to the staging API in batches, fetching a
// client-credentials token when an identity provider is configured.
type Deliverer struct {
	stagingURL   string
	tokenURL     string
	clientID     string
	clientSecret string
	http         *http.Client

	token       string
	tokenExpiry time.Time
}

func NewDeliverer(stagingURL, tokenURL, clientID, clientSecret string) *Deliverer {
	return &Deliverer{
		stagingURL:   stagingURL,
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         &http.Client{Timeout: 30 * time.Second},
	}
}

// Outcome mirrors the staging API's per-record result.
type Outcome struct {
	SourceRef  string `json:"sourceRef"`
	SourceHash string `json:"sourceHash"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	EntryID    string `json:"entryId"`
}

const deliveryBatchSize = 200

// Deliver submits records in batches and returns every outcome.
func (d *Deliverer) Deliver(ctx context.Context, records []Record) ([]Outcome, error) {
	outcomes := make([]Outcome, 0, len(records))

	for start := 0; start < len(records); start += deliveryBatchSize {
		end := min(start+deliveryBatchSize, len(records))

		batch, err := d.deliverBatch(ctx, records[start:end])
		if err != nil {
			return outcomes, err
		}
		outcomes = append(outcomes, batch...)
	}

	return outcomes, nil
}

func (d *Deliverer) deliverBatch(ctx context.Context, records []Record) ([]Outcome, error) {
	payload, err := json.Marshal(map[string]any{"records": records})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.stagingURL+"/v1/staging-records", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	if d.tokenURL != "" {
		token, err := d.accessToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching token: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := d.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("staging api unreachable: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("staging api returned %d: %s", response.StatusCode, string(body))
	}

	var parsed struct {
		Outcomes []Outcome `json:"outcomes"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parsing staging response: %w", err)
	}

	return parsed.Outcomes, nil
}

func (d *Deliverer) accessToken(ctx context.Context) (string, error) {
	if d.token != "" && time.Now().Before(d.tokenExpiry) {
		return d.token, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {d.clientID},
		"client_secret": {d.clientSecret},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := d.http.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d: %s", response.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.AccessToken == "" {
		return "", fmt.Errorf("token endpoint returned no access token")
	}

	d.token = parsed.AccessToken
	d.tokenExpiry = time.Now().Add(time.Duration(max(parsed.ExpiresIn-30, 30)) * time.Second)

	return d.token, nil
}
