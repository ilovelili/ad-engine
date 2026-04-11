package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ilovelili/ad-engine/internal/domain"
	"github.com/ilovelili/ad-engine/internal/store"
)

type ConnectPlatformRequest struct {
	Platform                   string `json:"platform"`
	AccountLabel               string `json:"accountLabel"`
	AccountIdentifier          string `json:"accountIdentifier"`
	Secret                     string `json:"secret"`
	InstagramBusinessAccountID string `json:"instagramBusinessAccountId"`
}

type PlatformConnector interface {
	Platform() string
	Connect(ctx context.Context, req ConnectPlatformRequest) (*PlatformConnectionResult, error)
}

type PlatformConnectionResult struct {
	AccountLabel      string
	AccountIdentifier string
	ExternalAccountID string
	DisplayName       string
	Scopes            []string
	AdAccounts        []domain.AdAccount
	Metadata          domain.PlatformConnectionMetadata
}

type PlatformConnectionService struct {
	store      *store.Store
	sealer     *CredentialSealer
	connectors map[string]PlatformConnector
}

func NewPlatformConnectionService(st *store.Store, sealer *CredentialSealer, connectors ...PlatformConnector) *PlatformConnectionService {
	registry := make(map[string]PlatformConnector, len(connectors))
	for _, connector := range connectors {
		registry[connector.Platform()] = connector
	}

	return &PlatformConnectionService{
		store:      st,
		sealer:     sealer,
		connectors: registry,
	}
}

func (s *PlatformConnectionService) SupportedPlatforms() []domain.SupportedPlatform {
	return []domain.SupportedPlatform{
		{
			ID:                  domain.PlatformInstagram,
			Name:                "Instagram",
			AuthenticationModel: "Meta OAuth",
			Fields:              []string{"oauth"},
		},
	}
}

func (s *PlatformConnectionService) List() (*domain.PlatformConnectionsView, error) {
	connections, err := s.store.ListPlatformConnections()
	if err != nil {
		return nil, err
	}

	snapshots := make([]domain.PlatformConnectionSnapshot, 0, len(connections))
	for _, connection := range connections {
		snapshots = append(snapshots, store.BuildConnectionSnapshot(connection))
	}

	return &domain.PlatformConnectionsView{
		SupportedPlatforms: s.SupportedPlatforms(),
		Connections:        snapshots,
	}, nil
}

func (s *PlatformConnectionService) Connect(ctx context.Context, req ConnectPlatformRequest) (*domain.PlatformConnectionSnapshot, error) {
	req.Platform = strings.TrimSpace(strings.ToLower(req.Platform))
	req.AccountIdentifier = strings.TrimSpace(req.AccountIdentifier)
	req.AccountLabel = strings.TrimSpace(req.AccountLabel)
	req.Secret = strings.TrimSpace(req.Secret)
	req.InstagramBusinessAccountID = strings.TrimSpace(req.InstagramBusinessAccountID)

	if req.Platform == "" {
		return nil, errors.New("platform is required")
	}
	if req.Secret == "" {
		return nil, errors.New("access token is required")
	}

	connector, ok := s.connectors[req.Platform]
	if !ok {
		return nil, fmt.Errorf("platform %q is not supported yet", req.Platform)
	}

	result, err := connector.Connect(ctx, req)
	if err != nil {
		return nil, err
	}

	nonce, ciphertext, err := s.sealer.Seal(req.Secret)
	if err != nil {
		return nil, err
	}

	metadataJSON, err := json.Marshal(result.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	adAccountsJSON, err := json.Marshal(result.AdAccounts)
	if err != nil {
		return nil, fmt.Errorf("marshal ad accounts: %w", err)
	}

	now := time.Now()
	connection := &domain.PlatformConnection{
		Platform:             req.Platform,
		AccountLabel:         firstNonEmpty(result.AccountLabel, req.AccountLabel, result.DisplayName),
		AccountIdentifier:    firstNonEmpty(result.AccountIdentifier, req.AccountIdentifier),
		ExternalAccountID:    result.ExternalAccountID,
		Status:               "connected",
		CredentialNonce:      base64.StdEncoding.EncodeToString(nonce),
		CredentialCiphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Scopes:               strings.Join(sortedUnique(result.Scopes), ","),
		LastValidatedAt:      &now,
		LastSyncAt:           &now,
		LastError:            "",
		MetadataJSON:         string(metadataJSON),
		AdAccountsJSON:       string(adAccountsJSON),
	}

	if err := s.store.SavePlatformConnection(connection); err != nil {
		return nil, err
	}

	snapshot := store.BuildConnectionSnapshot(*connection)
	return &snapshot, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}

	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
