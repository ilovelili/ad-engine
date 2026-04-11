# ad-engine

POC for an automated ad-delivery engine that:

- simulates cross-posting ads to X, TikTok, and Instagram
- tracks channel performance
- dynamically rebalances the budget allocation by performance
- stores platform connections and fetches available Meta ad accounts for Instagram
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

Optional environment variables for platform connections:

```bash
export CONNECTION_SECRET="replace-with-a-stable-secret"
export META_GRAPH_BASE_URL="https://graph.facebook.com"
export META_GRAPH_API_VERSION="v22.0"
export META_APP_ID="your-meta-app-id"
export META_APP_SECRET="your-meta-app-secret"
export META_REDIRECT_URI="http://localhost:8080/api/v1/oauth/meta/callback"
export META_OAUTH_SCOPES="ads_management,business_management"
```

For local development, copy `.env.example` to `.env` and fill in your Meta app credentials:

```bash
cp .env.example .env
```

For the local test page:

```bash
make test-page
```

Then open `http://localhost:8080` and use the `Connect with Meta` button.

## API

- `GET /api/v1/healthz`
- `GET /api/v1/dashboard`
- `POST /api/v1/rebalance`
- `GET /api/v1/connections`
- `GET /api/v1/oauth/meta/start`
- `GET /api/v1/oauth/meta/callback`

## Notes

- Redis is optional in this POC. If Redis is unavailable at `REDIS_ADDR`, the service still runs.
- The optimizer is intentionally simple: it uses simulated spend, CTR, conversion rate, and ROAS to reassign budget percentages every cycle.
- The Instagram connection flow now uses Meta OAuth, then exchanges the authorization code for a Meta access token server-side.
