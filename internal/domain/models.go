package domain

import "time"

type Campaign struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Status      string
	Goal        string
	TotalBudget float64
	Currency    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PlatformAllocation struct {
	ID            uint `gorm:"primaryKey"`
	CampaignID    uint `gorm:"index"`
	Platform      string
	AllocationPct float64
	Spend         float64
	Impressions   int64
	Clicks        int64
	Conversions   int64
	Revenue       float64
	PublishedAds  int64
	LastSyncedAt  time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DeliveryEvent struct {
	ID         uint `gorm:"primaryKey"`
	CampaignID uint `gorm:"index"`
	Platform   string
	Status     string
	Message    string
	CreatedAt  time.Time
}

type PlatformSnapshot struct {
	Platform      string    `json:"platform"`
	AllocationPct float64   `json:"allocationPct"`
	Spend         float64   `json:"spend"`
	Impressions   int64     `json:"impressions"`
	Clicks        int64     `json:"clicks"`
	Conversions   int64     `json:"conversions"`
	Revenue       float64   `json:"revenue"`
	ROAS          float64   `json:"roas"`
	CTR           float64   `json:"ctr"`
	PublishedAds  int64     `json:"publishedAds"`
	LastSyncedAt  time.Time `json:"lastSyncedAt"`
}

type CampaignSnapshot struct {
	CampaignID    uint               `json:"campaignId"`
	Name          string             `json:"name"`
	Status        string             `json:"status"`
	Goal          string             `json:"goal"`
	TotalBudget   float64            `json:"totalBudget"`
	Remaining     float64            `json:"remaining"`
	Currency      string             `json:"currency"`
	LastRebalance time.Time          `json:"lastRebalance"`
	Platforms     []PlatformSnapshot `json:"platforms"`
}
