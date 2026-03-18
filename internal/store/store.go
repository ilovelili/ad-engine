package store

import (
	"errors"
	"fmt"
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
