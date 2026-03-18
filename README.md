# ad-engine

POC for an automated ad-delivery engine that:

- simulates cross-posting ads to X, TikTok, and Instagram
- tracks channel performance
- dynamically rebalances the budget allocation by performance
- exposes a REST API and serves a lightweight dashboard

## Stack

- Go
- Echo
- GORM with SQLite for local persistence
- Redigo for caching dashboard snapshots in Redis when available

## Run

```bash
go run ./cmd/server
```

Then open `http://localhost:8080`.

## API

- `GET /api/v1/healthz`
- `GET /api/v1/dashboard`
- `POST /api/v1/rebalance`

## Notes

- Redis is optional in this POC. If Redis is unavailable at `REDIS_ADDR`, the service still runs.
- The optimizer is intentionally simple: it uses simulated spend, CTR, conversion rate, and ROAS to reassign budget percentages every cycle.
