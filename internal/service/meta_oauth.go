package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type MetaOAuthService struct {
	appID        string
	appSecret    string
	redirectURI  string
	graphBaseURL string
	apiVersion   string
	scopes       []string
	httpClient   *http.Client
}

func NewMetaOAuthService(appID, appSecret, redirectURI, graphBaseURL, apiVersion, scopes string) *MetaOAuthService {
	return &MetaOAuthService{
		appID:        strings.TrimSpace(appID),
		appSecret:    strings.TrimSpace(appSecret),
		redirectURI:  strings.TrimSpace(redirectURI),
		graphBaseURL: strings.TrimRight(strings.TrimSpace(graphBaseURL), "/"),
		apiVersion:   strings.Trim(strings.TrimSpace(apiVersion), "/"),
		scopes:       parseScopes(scopes),
		httpClient:   &http.Client{Timeout: 12 * time.Second},
	}
}

func (s *MetaOAuthService) Enabled() bool {
	return s.appID != "" && s.appSecret != "" && s.redirectURI != ""
}

func (s *MetaOAuthService) AuthorizeURL(state string) (string, error) {
	if !s.Enabled() {
		return "", errors.New("Meta OAuth is not configured")
	}

	query := url.Values{
		"client_id":     []string{s.appID},
		"redirect_uri":  []string{s.redirectURI},
		"state":         []string{state},
		"response_type": []string{"code"},
	}
	if len(s.scopes) > 0 {
		query.Set("scope", strings.Join(s.scopes, ","))
	}

	return fmt.Sprintf("https://www.facebook.com/%s/dialog/oauth?%s", s.apiVersion, query.Encode()), nil
}

func (s *MetaOAuthService) ExchangeCode(ctx context.Context, code string) (string, error) {
	if !s.Enabled() {
		return "", errors.New("Meta OAuth is not configured")
	}

	query := url.Values{
		"client_id":     []string{s.appID},
		"redirect_uri":  []string{s.redirectURI},
		"client_secret": []string{s.appSecret},
		"code":          []string{code},
	}

	endpoint := fmt.Sprintf("%s/%s/oauth/access_token?%s", s.graphBaseURL, s.apiVersion, query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create token exchange request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send token exchange request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var graphErr graphErrorEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&graphErr); err == nil && graphErr.Error.Message != "" {
			return "", fmt.Errorf("%s (type=%s code=%d)", graphErr.Error.Message, graphErr.Error.Type, graphErr.Error.Code)
		}
		return "", fmt.Errorf("unexpected token exchange status: %s", resp.Status)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode token exchange response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", errors.New("Meta OAuth token exchange returned no access token")
	}
	return payload.AccessToken, nil
}

func GenerateOAuthState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func parseScopes(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
