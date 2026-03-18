package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ilovelili/ad-engine/internal/cache"
	"github.com/ilovelili/ad-engine/internal/domain"
	"github.com/ilovelili/ad-engine/internal/store"
)

type Engine struct {
	store          *store.Store
	cache          *cache.Cache
	optimizer      *Optimizer
	rebalanceEvery time.Duration
	lastRebalance  time.Time
}

func NewEngine(st *store.Store, c *cache.Cache, rebalanceEvery time.Duration) *Engine {
	return &Engine{
		store:          st,
		cache:          c,
		optimizer:      NewOptimizer(),
		rebalanceEvery: rebalanceEvery,
		lastRebalance:  time.Now(),
	}
}

func (e *Engine) Start(ctx context.Context) {
	ticker := time.NewTicker(e.rebalanceEvery)
	defer ticker.Stop()

	if err := e.RunCycle(); err != nil {
		log.Printf("initial engine cycle error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.RunCycle(); err != nil {
				log.Printf("engine cycle error: %v", err)
			}
		}
	}
}

func (e *Engine) RunCycle() error {
	campaign, err := e.store.ActiveCampaign()
	if err != nil {
		return fmt.Errorf("load active campaign: %w", err)
	}
	if campaign == nil {
		return nil
	}

	allocations, err := e.store.AllocationsByCampaign(campaign.ID)
	if err != nil {
		return fmt.Errorf("load allocations: %w", err)
	}

	allocations, events := e.optimizer.SimulateTick(*campaign, allocations)
	allocations = e.optimizer.Rebalance(allocations)
	if err := ValidateAllocations(allocations); err != nil {
		return err
	}

	if err := e.store.SaveAllocations(allocations); err != nil {
		return fmt.Errorf("save allocations: %w", err)
	}
	if err := e.store.AddDeliveryEvents(events); err != nil {
		return fmt.Errorf("save delivery events: %w", err)
	}

	e.lastRebalance = time.Now()
	snapshot := BuildSnapshot(*campaign, allocations, e.lastRebalance)
	if err := e.cache.SetDashboard(snapshot); err != nil {
		log.Printf("cache dashboard snapshot: %v", err)
	}

	return nil
}

func (e *Engine) Dashboard() (*domain.CampaignSnapshot, error) {
	if snapshot, err := e.cache.GetDashboard(); err == nil && snapshot != nil {
		return snapshot, nil
	}

	campaign, err := e.store.ActiveCampaign()
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, nil
	}

	allocations, err := e.store.AllocationsByCampaign(campaign.ID)
	if err != nil {
		return nil, err
	}

	snapshot := BuildSnapshot(*campaign, allocations, e.lastRebalance)
	return &snapshot, nil
}
