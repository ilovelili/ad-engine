package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ilovelili/ad-engine/internal/domain"
)

type Store struct {
	db *gorm.DB
}

func New(dsn string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(
		&domain.Campaign{},
		&domain.PlatformAllocation{},
		&domain.DeliveryEvent{},
		&domain.PlatformConnection{},
	); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Seed() error {
	var count int64
	if err := s.db.Model(&domain.Campaign{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	campaign := domain.Campaign{
		Name:        "Q2 Growth Booster",
		Status:      "active",
		Goal:        "maximize conversions",
		TotalBudget: 10000,
		Currency:    "USD",
	}
	if err := s.db.Create(&campaign).Error; err != nil {
		return err
	}

	allocations := []domain.PlatformAllocation{
		{CampaignID: campaign.ID, Platform: "x", AllocationPct: 30, LastSyncedAt: time.Now()},
		{CampaignID: campaign.ID, Platform: "tiktok", AllocationPct: 40, LastSyncedAt: time.Now()},
		{CampaignID: campaign.ID, Platform: "instagram", AllocationPct: 30, LastSyncedAt: time.Now()},
	}

	return s.db.Create(&allocations).Error
}

func (s *Store) ActiveCampaign() (*domain.Campaign, error) {
	var campaign domain.Campaign
	err := s.db.Where("status = ?", "active").First(&campaign).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &campaign, err
}

func (s *Store) AllocationsByCampaign(campaignID uint) ([]domain.PlatformAllocation, error) {
	var allocations []domain.PlatformAllocation
	err := s.db.Where("campaign_id = ?", campaignID).Order("platform asc").Find(&allocations).Error
	return allocations, err
}

func (s *Store) SaveAllocations(allocations []domain.PlatformAllocation) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, allocation := range allocations {
			if err := tx.Save(&allocation).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) AddDeliveryEvents(events []domain.DeliveryEvent) error {
	if len(events) == 0 {
		return nil
	}
	return s.db.Create(&events).Error
}

func (s *Store) ListPlatformConnections() ([]domain.PlatformConnection, error) {
	var connections []domain.PlatformConnection
	err := s.db.Order("platform asc, account_label asc, account_identifier asc").Find(&connections).Error
	return connections, err
}

func (s *Store) SavePlatformConnection(connection *domain.PlatformConnection) error {
	var existing domain.PlatformConnection
	err := s.db.Where("platform = ? AND account_identifier = ?", connection.Platform, connection.AccountIdentifier).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.Create(connection).Error
	}
	if err != nil {
		return err
	}

	existing.AccountLabel = connection.AccountLabel
	existing.ExternalAccountID = connection.ExternalAccountID
	existing.Status = connection.Status
	existing.CredentialNonce = connection.CredentialNonce
	existing.CredentialCiphertext = connection.CredentialCiphertext
	existing.Scopes = connection.Scopes
	existing.LastValidatedAt = connection.LastValidatedAt
	existing.LastSyncAt = connection.LastSyncAt
	existing.LastError = connection.LastError
	existing.MetadataJSON = connection.MetadataJSON
	existing.AdAccountsJSON = connection.AdAccountsJSON

	if err := s.db.Save(&existing).Error; err != nil {
		return err
	}

	connection.ID = existing.ID
	connection.CreatedAt = existing.CreatedAt
	connection.UpdatedAt = existing.UpdatedAt
	return nil
}

func BuildConnectionSnapshot(connection domain.PlatformConnection) domain.PlatformConnectionSnapshot {
	snapshot := domain.PlatformConnectionSnapshot{
		ID:                connection.ID,
		Platform:          connection.Platform,
		AccountLabel:      connection.AccountLabel,
		AccountIdentifier: connection.AccountIdentifier,
		ExternalAccountID: connection.ExternalAccountID,
		Status:            connection.Status,
		Scopes:            splitCSV(connection.Scopes),
		LastValidatedAt:   connection.LastValidatedAt,
		LastSyncAt:        connection.LastSyncAt,
		LastError:         connection.LastError,
		AdAccounts:        []domain.AdAccount{},
	}

	if connection.MetadataJSON != "" {
		var metadata domain.PlatformConnectionMetadata
		if err := json.Unmarshal([]byte(connection.MetadataJSON), &metadata); err == nil {
			snapshot.DisplayName = metadata.DisplayName
			snapshot.InstagramBusinessAccountID = metadata.InstagramBusinessAccountID
		}
	}

	if connection.AdAccountsJSON != "" {
		var adAccounts []domain.AdAccount
		if err := json.Unmarshal([]byte(connection.AdAccountsJSON), &adAccounts); err == nil {
			snapshot.AdAccounts = adAccounts
		}
	}

	return snapshot
}

func splitCSV(value string) []string {
	if value == "" {
		return []string{}
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
