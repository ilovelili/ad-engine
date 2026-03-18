# Ad Optimization Whitepaper

## Title

Adaptive Cross-Platform Ad Delivery and Budget Reallocation for X, TikTok, and Instagram

## Version

- Version: `0.2`
- Date: `2026-03-18`
- Scope: POC design for `ad-engine`

## 1. Executive Summary

This whitepaper defines the optimization algorithm for `ad-engine`, a proof-of-concept system that:

- cross-posts ads to X, TikTok, and Instagram
- collects normalized performance signals from each platform
- dynamically rebalances budget percentages across platforms
- exposes the current allocation and performance state through a REST API and dashboard

The core design goal is to maximize business outcome under uncertainty. In practice, that means the system must avoid a naive "send all budget to the best recent performer" strategy, because performance is noisy, delayed, and platform-specific. Instead, the optimizer should:

- balance exploration and exploitation
- pace spend against campaign budget limits
- account for delayed conversions
- maintain minimum spend on each platform to prevent premature starvation
- keep a human approval boundary around high-risk actions

For the POC, we recommend a constrained multi-armed bandit approach combined with portfolio-style rebalancing ideas borrowed from cross-token exchange systems. This gives us a strong path from a simple heuristic implementation to a more robust adaptive optimizer without changing the system architecture.

## 2. Problem Statement

In a cross-platform advertising workflow, each platform reports different metrics, at different cadences, with different auction dynamics:

- X may react quickly but with noisier conversion quality
- TikTok may deliver stronger discovery and cheaper impressions
- Instagram may perform differently for branded or visual creative

A practical optimizer must answer one recurring question:

How should the next unit of budget be split across platforms to maximize expected return while managing uncertainty and delivery risk?

This becomes harder because:

- attribution can be delayed
- reported revenue may lag spend
- conversion quality is not identical across platforms
- a platform can look temporarily strong due to low sample size
- budget changes too frequently can destabilize learning

## 3. Product Context

The current `ad-engine` POC consists of:

- a Go REST API server
- persistence through GORM
- optional Redis caching through Redigo
- a dashboard showing current allocation and performance
- a simulated delivery loop that posts to X, TikTok, and Instagram

The current implementation already includes a simple score-based rebalancer. This whitepaper proposes the next algorithmic shape that should guide future implementation.

Relevant code today:

- [internal/service/optimizer.go](/Users/min/ad/Projects/src/github.com/ilovelili/ad-engine/internal/service/optimizer.go)
- [internal/service/engine.go](/Users/min/ad/Projects/src/github.com/ilovelili/ad-engine/internal/service/engine.go)
- [internal/domain/models.go](/Users/min/ad/Projects/src/github.com/ilovelili/ad-engine/internal/domain/models.go)

## 4. Reference Market Signal

A recent PR TIMES announcement from renue describes an "advertising agency AI agent" that spans campaign design, creative production, cross-platform delivery, monitoring, and budget reallocation across major ad platforms including X, TikTok, and Instagram/Meta. It also explicitly keeps human approval in the loop for critical decisions and acknowledges legal and policy constraints.

Source:

- PR TIMES, 2026-02-19: [renue、「広告代理AIエージェント」を開発、Web広告のクリエイティブ制作、配信、運用まで、生成AIが完全代行](https://prtimes.jp/main/html/rd/p/000000022.000091210.html)

Inference from that reference:

- the competitive direction is not just automatic bidding, but an orchestration layer over multiple ad APIs
- budget reallocation is only one part of the system, but it is central to performance differentiation
- human approval and compliance boundaries should be first-class parts of the design

Our POC should therefore prioritize a credible optimization layer and clean decision logging, even before deep creative automation.

## 4.1 Borrowing From Cross-Token Rebalancing

We should explicitly borrow ideas from cross-platform token exchange rebalancing.

Why this analogy is useful:

- token portfolio systems decide how capital should be redistributed across venues or assets
- ad optimization systems decide how budget should be redistributed across platforms
- both systems must account for execution friction, overreaction risk, and confidence in recent performance

Concept mapping:

- portfolio weight maps to platform budget percentage
- expected return maps to expected conversion or ROAS value
- slippage maps to performance degradation when budget changes too quickly
- liquidity depth maps to audience capacity and auction absorption
- transaction cost maps to platform learning-reset cost and creative fatigue
- rebalance threshold maps to the minimum drift that justifies moving budget

This borrowing is intentional, not accidental. The optimizer in `ad-engine` should behave less like a raw score sorter and more like a constrained portfolio allocator.

Important difference:

- token markets provide much faster execution feedback
- ad systems have delayed attribution, noisier reward signals, and policy constraints

So we borrow the rebalancing structure, but we do not assume market-like immediacy.

## 5. Optimization Objectives

The optimizer should support a configurable primary objective:

- maximize `conversions`
- maximize `revenue`
- minimize `CPA`
- maximize `ROAS`

For the first POC phase, the default objective should be:

- maximize expected conversions subject to ROAS and pacing constraints

Secondary objectives:

- maintain budget pacing against total campaign budget
- avoid overreaction to sparse or noisy data
- preserve platform learning with a minimum allocation floor
- provide explanations for each rebalance decision

## 6. Design Principles

### 6.1 Normalize Before Comparing

Raw platform metrics are not directly comparable. We should compare derived normalized features such as:

- CTR
- CVR
- CPC
- CPA
- ROAS
- spend velocity
- impression velocity

### 6.2 Separate Signal From Policy

The optimizer should produce an unconstrained platform score first. Then policy constraints should shape the final allocation:

- min allocation floor
- max allocation cap
- max step-up per cycle
- max step-down per cycle
- pacing constraints

### 6.3 Reward Confidence, Not Just Performance

A platform with 2 conversions on tiny spend should not immediately dominate one with 200 conversions on meaningful spend. Confidence weighting is required.

### 6.4 Keep Rebalance Intervals Stable

Rebalancing too frequently causes thrashing. For the POC:

- metric ingestion can be frequent
- budget reallocation should happen on a slower cadence, such as every 30 to 60 minutes in production, or every few seconds in simulation

This is the same logic used in threshold-based portfolio rebalancing: do not trade unless the drift is meaningful enough to justify the move.

### 6.5 Human Approval for High-Risk Actions

The system should require approval for:

- launching a new platform
- increasing total budget
- pushing creative that has not passed review
- violating policy or brand-safety constraints

## 7. Proposed Algorithm

## 7.1 Overview

We recommend a 5-stage decision pipeline:

1. Ingest and normalize per-platform metrics
2. Estimate each platform's expected utility and confidence
3. Convert utility into target allocation using constrained bandit logic
4. Apply portfolio-style rebalance threshold and reallocation friction
5. Apply pacing and safety rules, then publish the new budget split

## 7.2 Input Metrics

Per platform `p`, for a rolling window `W`, collect:

- `spend_p`
- `impressions_p`
- `clicks_p`
- `conversions_p`
- `revenue_p`
- `published_ads_p`
- `last_synced_at_p`

Derived metrics:

- `ctr_p = clicks_p / impressions_p`
- `cvr_p = conversions_p / clicks_p`
- `cpc_p = spend_p / clicks_p`
- `cpa_p = spend_p / conversions_p`
- `roas_p = revenue_p / spend_p`

Stability metrics:

- `sample_weight_p = log(1 + impressions_p)`
- `conversion_weight_p = log(1 + conversions_p)`
- `recency_weight_p = exp(-lambda * age_minutes)`

## 7.3 Utility Score

For a conversions-first objective, define:

```text
raw_score_p =
  w_roas * norm(roas_p) +
  w_cvr  * norm(cvr_p)  +
  w_ctr  * norm(ctr_p)  -
  w_cpa  * norm(cpa_p)
```

Where `norm()` is a rolling z-score or min-max normalization across active platforms.

Then apply confidence:

```text
confidence_p =
  min(1.0, a * sample_weight_p + b * conversion_weight_p)

blended_score_p =
  confidence_p * raw_score_p + (1 - confidence_p) * prior_score_p
```

`prior_score_p` is a prior belief for each platform. In early POC phases, it can be a neutral default like `0.5`, or a platform prior based on historical campaigns.

## 7.4 Exploration Bonus

To avoid locking onto a local optimum too early, add an uncertainty bonus:

```text
exploration_bonus_p = c * sqrt(ln(T + 1) / (n_p + 1))
```

Where:

- `T` is total decision rounds
- `n_p` is effective number of times platform `p` has received meaningful spend
- `c` controls exploration

Final decision score:

```text
decision_score_p = blended_score_p + exploration_bonus_p
```

This is a UCB-style bandit term. For production later, Thompson Sampling would also be a strong choice, but UCB is easier to explain and debug in an early-stage system.

## 7.5 Convert Score to Allocation

Transform scores into target percentages with softmax:

```text
target_share_p = exp(decision_score_p / tau) / sum(exp(decision_score_k / tau))
```

Where `tau` is a temperature parameter:

- lower `tau` makes the optimizer more aggressive
- higher `tau` keeps the distribution flatter

After softmax, convert the raw weights into constrained target shares with:

- minimum allocation floor
- maximum allocation cap
- redistribution of excess weight from capped platforms

This matches portfolio construction logic where target weights are bounded by policy constraints.

## 7.6 Rebalance Threshold and Execution Friction

The system should not move budget on every small ranking change.

Define:

```text
drift_p = |target_share_p - current_share_p|
max_drift = max(drift_p)
```

If `max_drift < threshold`, keep the current allocation unchanged.

Recommended default:

- `threshold = 5 allocation points`

When a rebalance is justified, apply execution friction:

```text
friction = max(floor_friction, 1 - k * max_drift)
```

Then smooth the move:

```text
new_share_p =
  current_share_p +
  gamma * friction * scaled_delta_p
```

Where:

- `scaled_delta_p` is the target delta after a max-step constraint
- `gamma` is a smoothing factor

This is directly inspired by cross-token rebalancing systems where the model must weigh target optimality against the cost of moving capital.

## 7.7 Apply Business Constraints

After deriving `target_share_p`, enforce:

- minimum platform floor: `15%`
- maximum platform cap: `60%`
- maximum upward move per cycle: `+10 points`
- maximum downward move per cycle: `-10 points`
- total shares must sum to `100%`

These values are suitable defaults for the POC and should be configurable.

## 7.8 Budget Pacing Layer

The optimizer should not only choose shares; it should also decide how fast to spend.

Define:

- `expected_spend_by_now`
- `actual_spend_by_now`
- `pacing_ratio = actual / expected`

Rules:

- if `pacing_ratio > 1.1`, reduce aggressiveness and hold more budget
- if `pacing_ratio < 0.9`, allow stronger reallocation toward high-performing channels

This separates "where budget should go" from "how fast the campaign should spend."

## 7.9 Delayed Conversion Handling

Some platforms or products have delayed conversion attribution. To avoid penalizing those channels too early:

- maintain both short-window and long-window metrics
- weight recent engagement metrics more heavily when conversion data is sparse
- use lag-adjusted expected conversion estimates

Example:

```text
effective_conversion_signal_p =
  alpha * short_window_cvr_p +
  (1 - alpha) * long_window_cvr_p
```

This is especially important if TikTok or Instagram upper-funnel activity converts later through branded search or direct traffic.

## 8. Recommended POC Formula

For the next implementation step in `ad-engine`, we recommend the following simple but credible formula:

```text
score_p =
  0.45 * norm(roas_p) +
  0.25 * norm(cvr_p) +
  0.15 * norm(ctr_p) -
  0.15 * norm(cpa_p)

confidence_p =
  min(1.0, 0.15 * log(1 + impressions_p) + 0.25 * log(1 + conversions_p))

decision_score_p =
  confidence_p * score_p +
  (1 - confidence_p) * prior_p +
  0.2 * sqrt(ln(T + 1) / (n_p + 1))
```

Then:

- apply softmax
- enforce floors and caps
- rebalance only when drift is above threshold
- apply friction and smoothing against the previous allocation

Allocation smoothing:

```text
new_allocation_p =
  (1 - gamma) * current_allocation_p + gamma * constrained_target_p
```

Recommended `gamma`:

- `0.2` to `0.35` for stable production behavior
- larger values for simulation demos

Rebalance threshold:

- `5 points` is a strong default for the POC

Friction interpretation:

- larger planned moves should be damped because the act of reallocating itself creates execution cost

## 9. Decision Explainability

Every rebalance event should produce a machine-readable explanation:

```json
{
  "platform": "tiktok",
  "previousAllocationPct": 34.0,
  "newAllocationPct": 41.5,
  "topDrivers": [
    "highest normalized ROAS in last window",
    "strong conversion rate confidence",
    "campaign pacing under target allows expansion"
  ],
  "guardrailsApplied": [
    "max_step_up_per_cycle"
  ]
}
```

This should be stored as part of a rebalance log. Explainability matters because operators need to know whether a shift came from real performance, sparse data, or a safety rule.

## 10. Dashboard Requirements

The dashboard should show:

- current allocation percentage by platform
- spend, clicks, conversions, revenue
- CTR, CVR, CPA, ROAS
- current pacing status
- last rebalance timestamp
- reason summary for the most recent budget shift

Future additions:

- confidence score by platform
- exploration vs exploitation indicator
- drift or anomaly warning

## 11. Data Model Extensions

The current schema should be extended over time with:

- `rebalance_events`
- `metric_snapshots`
- `platform_priors`
- `creative_variants`
- `approval_events`

Suggested `rebalance_events` fields:

- `campaign_id`
- `platform`
- `previous_allocation_pct`
- `new_allocation_pct`
- `decision_score`
- `confidence_score`
- `exploration_bonus`
- `explanation_json`
- `created_at`

## 12. Safety, Policy, and Compliance

The optimization engine must not be treated as a fully autonomous business actor. Required controls:

- hard campaign budget ceiling
- allowlist of supported platforms and campaign types
- human review for ad copy and creative before first publication
- audit trail for budget changes
- policy compliance checks for platform-specific ad rules
- legal review for claims, disclosures, and regulated industries

This aligns with the market direction seen in the renue announcement, which publicly states that final approvals and budget ceilings remain under human control.

## 13. POC-to-Production Roadmap

### Phase 1: Heuristic Optimizer

- use current score-based logic
- add normalized metrics
- log rebalance explanations
- keep fixed floors and caps

### Phase 2: Constrained UCB Bandit

- add exploration bonus
- introduce confidence-weighted scoring
- add pacing-aware allocation adjustment

### Phase 3: Creative and Audience Feedback Loop

- optimize not only by platform, but also by creative variant
- learn cross-platform message transfer
- incorporate audience segment effects

### Phase 4: Full Decisioning Layer

- delayed attribution modeling
- anomaly detection
- forecast-based budget planning
- approval workflow and rollback controls

## 14. Implementation Recommendation For This Repo

The next concrete change to `ad-engine` should be:

1. Extend the optimizer in [internal/service/optimizer.go](/Users/min/ad/Projects/src/github.com/ilovelili/ad-engine/internal/service/optimizer.go) to compute normalized `CTR`, `CVR`, `CPA`, and `ROAS`.
2. Add confidence weighting and smoothing to prevent sudden allocation swings.
3. Add a `rebalance_events` model and persist explanation payloads.
4. Add dashboard fields for `CPA`, `CVR`, and rebalance reasons.
5. Keep the current mock publishers and simulation loop while the optimization logic matures.

## 15. Conclusion

The right algorithm for this product is not a single formula, but a decision stack:

- normalized multi-metric scoring
- confidence-aware exploration
- threshold-based portfolio rebalancing
- reallocation friction modeled after capital movement cost
- constrained allocation conversion
- pacing and compliance guardrails

For this POC, a constrained UCB-style bandit with smoothing is the best balance of:

- simplicity
- explainability
- extensibility
- practical implementation effort in Go

Borrowing from cross-token rebalancing makes the optimizer meaningfully better: it introduces disciplined movement thresholds and cost-aware reallocation behavior. That gives `ad-engine` a more credible optimization core today while leaving a clean path toward more advanced production-grade decisioning.
