package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ilovelili/ad-engine/internal/domain"
)

type InstagramConnector struct {
	baseURL    string
	apiVersion string
	httpClient *http.Client
}

func NewInstagramConnector(baseURL, apiVersion string) *InstagramConnector {
	return &InstagramConnector{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiVersion: strings.Trim(strings.TrimSpace(apiVersion), "/"),
		httpClient: &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *InstagramConnector) Platform() string {
	return domain.PlatformInstagram
}

func (c *InstagramConnector) Connect(ctx context.Context, req ConnectPlatformRequest) (*PlatformConnectionResult, error) {
	profile, err := c.fetchProfile(ctx, req.Secret)
	if err != nil {
		return nil, err
	}

	adAccounts, err := c.fetchAdAccounts(ctx, req.Secret)
	if err != nil {
		return nil, err
	}

	result := &PlatformConnectionResult{
		AccountLabel:      firstNonEmpty(req.AccountLabel, profile.Name, profile.ID),
		AccountIdentifier: firstNonEmpty(req.AccountIdentifier, profile.ID),
		ExternalAccountID: profile.ID,
		DisplayName:       firstNonEmpty(profile.Name, req.AccountIdentifier),
		Scopes: []string{
			"ads_management",
			"business_management",
		},
		AdAccounts: adAccounts,
		Metadata: domain.PlatformConnectionMetadata{
			DisplayName:                firstNonEmpty(profile.Name, req.AccountIdentifier),
			InstagramBusinessAccountID: req.InstagramBusinessAccountID,
		},
	}

	return result, nil
}

type graphProfileResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type graphAdAccountsResponse struct {
	Data []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		AccountStatus int    `json:"account_status"`
		Currency      string `json:"currency"`
		TimezoneName  string `json:"timezone_name"`
	} `json:"data"`
}

type graphErrorEnvelope struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func (c *InstagramConnector) fetchProfile(ctx context.Context, token string) (*graphProfileResponse, error) {
	endpoint := c.graphURL("me", url.Values{
		"fields":       []string{"id,name"},
		"access_token": []string{token},
	})

	var profile graphProfileResponse
	if err := c.getJSON(ctx, endpoint, &profile); err != nil {
		return nil, fmt.Errorf("validate Meta access token: %w", err)
	}
	return &profile, nil
}

func (c *InstagramConnector) fetchAdAccounts(ctx context.Context, token string) ([]domain.AdAccount, error) {
	endpoint := c.graphURL("me/adaccounts", url.Values{
		"fields":       []string{"id,name,account_status,currency,timezone_name"},
		"access_token": []string{token},
	})

	var response graphAdAccountsResponse
	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("fetch ad accounts from Meta API: %w", err)
	}

	adAccounts := make([]domain.AdAccount, 0, len(response.Data))
	for _, account := range response.Data {
		adAccounts = append(adAccounts, domain.AdAccount{
			ID:       account.ID,
			Name:     account.Name,
			Status:   metaAdAccountStatus(account.AccountStatus),
			Currency: account.Currency,
			Timezone: account.TimezoneName,
		})
	}
	return adAccounts, nil
}

func (c *InstagramConnector) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var graphErr graphErrorEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&graphErr); err == nil && graphErr.Error.Message != "" {
			return fmt.Errorf("%s (type=%s code=%d)", graphErr.Error.Message, graphErr.Error.Type, graphErr.Error.Code)
		}
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (c *InstagramConnector) graphURL(path string, query url.Values) string {
	return fmt.Sprintf("%s/%s/%s?%s", c.baseURL, c.apiVersion, strings.TrimLeft(path, "/"), query.Encode())
}

func metaAdAccountStatus(value int) string {
	switch value {
	case 1:
		return "active"
	case 2:
		return "disabled"
	case 3:
		return "unsettled"
	case 7:
		return "pending_risk_review"
	case 8:
		return "pending_settlement"
	case 9:
		return "in_grace_period"
	case 100:
		return "pending_closure"
	case 101:
		return "closed"
	default:
		return fmt.Sprintf("status_%d", value)
	}
}
