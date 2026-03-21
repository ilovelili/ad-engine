package service

import (
	"fmt"
	"math"
	"time"

	"github.com/ilovelili/ad-engine/internal/domain"
)

var qualityFactor = map[string]float64{
	"x":         0.95,
	"tiktok":    1.20,
	"instagram": 1.05,
}

type Optimizer struct {
	publishers map[string]Publisher
}

type optimizerConfig struct {
	minAllocation        float64
	maxAllocation        float64
	driftThreshold       float64
	maxStepPerCycle      float64
	smoothingFactor      float64
	temperature          float64
	explorationWeight    float64
	reallocationFriction float64
	priorScore           float64
}

type platformMetrics struct {
	currentAllocation float64
	roas              float64
	cvr               float64
	ctr               float64
	cpa               float64
	confidence        float64
	explorationBonus  float64
	score             float64
}

type deltaExecutionPlan struct {
	targetAllocations []float64
	maxDrift          float64
}

func NewOptimizer() *Optimizer {
	return &Optimizer{
		publishers: map[string]Publisher{
			"x":         NewMockPublisher("x"),
			"tiktok":    NewMockPublisher("tiktok"),
			"instagram": NewMockPublisher("instagram"),
		},
	}
}

func (o *Optimizer) SimulateTick(campaign domain.Campaign, allocations []domain.PlatformAllocation) ([]domain.PlatformAllocation, []domain.DeliveryEvent) {
	now := time.Now()
	events := make([]domain.DeliveryEvent, 0, len(allocations))

	for i := range allocations {
		allocation := &allocations[i]
		factor := qualityFactor[allocation.Platform]
		incrementSpend := campaign.TotalBudget * (allocation.AllocationPct / 100) * 0.015
		impressions := int64(math.Round(incrementSpend * 110 * factor))
		clicks := int64(math.Round(float64(impressions) * (0.018 + factor*0.003)))
		conversions := int64(math.Round(float64(clicks) * (0.05 + factor*0.02)))
		revenue := incrementSpend * (1.3 + factor*0.4)

		allocation.Spend += incrementSpend
		allocation.Impressions += impressions
		allocation.Clicks += clicks
		allocation.Conversions += conversions
		allocation.Revenue += revenue
		allocation.PublishedAds++
		allocation.LastSyncedAt = now

		if publisher, ok := o.publishers[allocation.Platform]; ok {
			events = append(events, publisher.Publish(campaign, *allocation))
		}
	}

	return allocations, events
}

func (o *Optimizer) Rebalance(allocations []domain.PlatformAllocation) []domain.PlatformAllocation {
	cfg := optimizerConfig{
		minAllocation:        15,
		maxAllocation:        60,
		driftThreshold:       5,
		maxStepPerCycle:      10,
		smoothingFactor:      0.65,
		temperature:          0.85,
		explorationWeight:    0.2,
		reallocationFriction: 0.03,
		priorScore:           0.5,
	}

	metrics := buildPlatformMetrics(allocations, cfg)
	plan := buildDeltaExecutionPlan(metrics, cfg)
	return executeDeltaRebalance(allocations, plan, cfg)
}

func BuildSnapshot(campaign domain.Campaign, allocations []domain.PlatformAllocation, lastRebalance time.Time) domain.CampaignSnapshot {
	platforms := make([]domain.PlatformSnapshot, 0, len(allocations))
	totalSpend := 0.0

	for _, allocation := range allocations {
		totalSpend += allocation.Spend
		platforms = append(platforms, domain.PlatformSnapshot{
			Platform:      allocation.Platform,
			AllocationPct: allocation.AllocationPct,
			Spend:         round2(allocation.Spend),
			Impressions:   allocation.Impressions,
			Clicks:        allocation.Clicks,
			Conversions:   allocation.Conversions,
			Revenue:       round2(allocation.Revenue),
			ROAS:          round2(ratio(allocation.Revenue, allocation.Spend)),
			CTR:           round2(ratio(float64(allocation.Clicks), float64(allocation.Impressions)) * 100),
			PublishedAds:  allocation.PublishedAds,
			LastSyncedAt:  allocation.LastSyncedAt,
		})
	}

	return domain.CampaignSnapshot{
		CampaignID:    campaign.ID,
		Name:          campaign.Name,
		Status:        campaign.Status,
		Goal:          campaign.Goal,
		TotalBudget:   round2(campaign.TotalBudget),
		Remaining:     round2(math.Max(campaign.TotalBudget-totalSpend, 0)),
		Currency:      campaign.Currency,
		LastRebalance: lastRebalance,
		Platforms:     platforms,
	}
}

func ratio(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func totalPublishedAds(allocations []domain.PlatformAllocation) int64 {
	var total int64
	for _, allocation := range allocations {
		total += allocation.PublishedAds
	}
	return total
}

func buildPlatformMetrics(allocations []domain.PlatformAllocation, cfg optimizerConfig) []platformMetrics {
	metrics := make([]platformMetrics, len(allocations))
	roasValues := make([]float64, len(allocations))
	cvrValues := make([]float64, len(allocations))
	ctrValues := make([]float64, len(allocations))
	cpaValues := make([]float64, len(allocations))
	totalRounds := float64(len(allocations)) + float64(totalPublishedAds(allocations)) + 1

	for i, allocation := range allocations {
		metrics[i] = platformMetrics{
			currentAllocation: allocation.AllocationPct,
			roas:              ratio(allocation.Revenue, allocation.Spend),
			cvr:               ratio(float64(allocation.Conversions), float64(allocation.Clicks)),
			ctr:               ratio(float64(allocation.Clicks), float64(allocation.Impressions)),
			cpa:               safeCPA(allocation.Spend, allocation.Conversions),
			confidence:        confidenceScore(allocation.Impressions, allocation.Conversions),
			explorationBonus:  cfg.explorationWeight * math.Sqrt(math.Log(totalRounds)/(float64(allocation.PublishedAds)+1)),
		}
		roasValues[i] = metrics[i].roas
		cvrValues[i] = metrics[i].cvr
		ctrValues[i] = metrics[i].ctr
		cpaValues[i] = metrics[i].cpa
	}

	normalizedROAS := normalizeSeries(roasValues)
	normalizedCVR := normalizeSeries(cvrValues)
	normalizedCTR := normalizeSeries(ctrValues)
	normalizedCPA := normalizeSeries(cpaValues)

	for i := range metrics {
		rawScore := (normalizedROAS[i] * 0.45) + (normalizedCVR[i] * 0.25) + (normalizedCTR[i] * 0.15) - (normalizedCPA[i] * 0.15)
		metrics[i].score = (metrics[i].confidence * rawScore) + ((1 - metrics[i].confidence) * cfg.priorScore) + metrics[i].explorationBonus
	}

	return metrics
}

func buildDeltaExecutionPlan(metrics []platformMetrics, cfg optimizerConfig) deltaExecutionPlan {
	decisionScores := make([]float64, len(metrics))
	for i, metric := range metrics {
		decisionScores[i] = metric.score
	}

	targets := constrainedTargets(softmax(decisionScores, cfg.temperature), cfg.minAllocation, cfg.maxAllocation)
	maxDrift := 0.0
	for i, target := range targets {
		drift := math.Abs(target - metrics[i].currentAllocation)
		if drift > maxDrift {
			maxDrift = drift
		}
	}

	return deltaExecutionPlan{
		targetAllocations: targets,
		maxDrift:          maxDrift,
	}
}

func executeDeltaRebalance(allocations []domain.PlatformAllocation, plan deltaExecutionPlan, cfg optimizerConfig) []domain.PlatformAllocation {
	// Delta-inspired execution: do nothing until the target/current gap is large enough
	// to justify moving budget across platforms.
	if plan.maxDrift < cfg.driftThreshold {
		return allocations
	}

	scale := math.Min(1, cfg.maxStepPerCycle/plan.maxDrift)
	friction := math.Max(0.25, 1-(cfg.reallocationFriction*plan.maxDrift))
	blend := cfg.smoothingFactor * friction * scale

	for i := range allocations {
		target := allocations[i].AllocationPct + ((plan.targetAllocations[i] - allocations[i].AllocationPct) * blend)
		target = applyBandwidthCap(allocations[i], target, cfg.maxStepPerCycle)
		allocations[i].AllocationPct = math.Round(target*10) / 10
		allocations[i].LastSyncedAt = time.Now()
	}

	normalizeAllocations(allocations, cfg.minAllocation, cfg.maxAllocation)
	return allocations
}

func applyBandwidthCap(allocation domain.PlatformAllocation, target, maxStepPerCycle float64) float64 {
	delta := target - allocation.AllocationPct
	if delta > maxStepPerCycle {
		delta = maxStepPerCycle
	}
	if delta < -maxStepPerCycle {
		delta = -maxStepPerCycle
	}

	// Treat recent delivery volume as a rough capacity signal. As a platform saturates,
	// damp upward movement to avoid over-allocating into a narrow audience pocket.
	capacityDampener := 1.0
	if allocation.PublishedAds > 0 {
		capacityDampener = math.Max(0.55, 1-(math.Log1p(float64(allocation.PublishedAds))*0.04))
	}
	if delta > 0 {
		delta *= capacityDampener
	}

	return allocation.AllocationPct + delta
}

func confidenceScore(impressions, conversions int64) float64 {
	score := (0.12 * math.Log1p(float64(impressions))) + (0.25 * math.Log1p(float64(conversions)))
	return math.Min(1, score)
}

func safeCPA(spend float64, conversions int64) float64 {
	if conversions == 0 {
		return spend + 1
	}
	return spend / float64(conversions)
}

func normalizeSeries(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	minValue := values[0]
	maxValue := values[0]
	for _, value := range values[1:] {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}

	normalized := make([]float64, len(values))
	if maxValue-minValue == 0 {
		for i := range normalized {
			normalized[i] = 0.5
		}
		return normalized
	}

	for i, value := range values {
		normalized[i] = (value - minValue) / (maxValue - minValue)
	}
	return normalized
}

func softmax(values []float64, temperature float64) []float64 {
	if temperature <= 0 {
		temperature = 1
	}

	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}

	result := make([]float64, len(values))
	total := 0.0
	for i, value := range values {
		scaled := math.Exp((value - maxValue) / temperature)
		result[i] = scaled
		total += scaled
	}

	for i := range result {
		result[i] /= total
	}
	return result
}

func constrainedTargets(weights []float64, minAllocation, maxAllocation float64) []float64 {
	targets := make([]float64, len(weights))
	remaining := 100 - (minAllocation * float64(len(weights)))
	for i, weight := range weights {
		targets[i] = minAllocation + (weight * remaining)
	}

	for {
		excess := 0.0
		activeWeight := 0.0
		active := make([]bool, len(targets))

		for i := range targets {
			if targets[i] > maxAllocation {
				excess += targets[i] - maxAllocation
				targets[i] = maxAllocation
				continue
			}
			active[i] = true
			activeWeight += weights[i]
		}

		if excess <= 0 || activeWeight == 0 {
			break
		}

		for i := range targets {
			if active[i] {
				targets[i] += excess * (weights[i] / activeWeight)
			}
		}
	}

	return targets
}

func normalizeAllocations(allocations []domain.PlatformAllocation, minAllocation, maxAllocation float64) {
	total := 0.0
	for i := range allocations {
		allocations[i].AllocationPct = math.Max(minAllocation, math.Min(maxAllocation, allocations[i].AllocationPct))
		total += allocations[i].AllocationPct
	}

	if total == 0 {
		return
	}

	diff := 100 - total
	for math.Abs(diff) > 0.01 {
		adjusted := false
		for i := range allocations {
			next := allocations[i].AllocationPct + diff
			if next < minAllocation || next > maxAllocation {
				continue
			}
			allocations[i].AllocationPct = math.Round(next*10) / 10
			adjusted = true
			break
		}
		if !adjusted {
			break
		}

		total = 0
		for _, allocation := range allocations {
			total += allocation.AllocationPct
		}
		diff = round2(100 - total)
	}
}

func ValidateAllocations(allocations []domain.PlatformAllocation) error {
	total := 0.0
	for _, allocation := range allocations {
		total += allocation.AllocationPct
	}
	if math.Abs(total-100) > 0.25 {
		return fmt.Errorf("allocation total must be close to 100, got %.2f", total)
	}
	return nil
}
