package service

import (
	"fmt"
	"time"

	"github.com/ilovelili/ad-engine/internal/domain"
)

type Publisher interface {
	Name() string
	Publish(campaign domain.Campaign, allocation domain.PlatformAllocation) domain.DeliveryEvent
}

type MockPublisher struct {
	platform string
}

func NewMockPublisher(platform string) MockPublisher {
	return MockPublisher{platform: platform}
}

func (p MockPublisher) Name() string {
	return p.platform
}

func (p MockPublisher) Publish(campaign domain.Campaign, allocation domain.PlatformAllocation) domain.DeliveryEvent {
	message := fmt.Sprintf(
		"Queued creative for %s with %.1f%% budget share",
		p.platform,
		allocation.AllocationPct,
	)

	return domain.DeliveryEvent{
		CampaignID: campaign.ID,
		Platform:   p.platform,
		Status:     "posted",
		Message:    message,
		CreatedAt:  time.Now(),
	}
}
