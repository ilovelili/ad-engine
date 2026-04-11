package app

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/ilovelili/ad-engine/internal/cache"
	"github.com/ilovelili/ad-engine/internal/config"
	httpapi "github.com/ilovelili/ad-engine/internal/http"
	"github.com/ilovelili/ad-engine/internal/service"
	"github.com/ilovelili/ad-engine/internal/store"
)

type App struct {
	store             *store.Store
	cache             *cache.Cache
	engine            *service.Engine
	connectionService *service.PlatformConnectionService
	metaOAuth         *service.MetaOAuthService
	handler           *httpapi.Handler
}

func New(cfg config.Config) (*App, error) {
	st, err := store.New(cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	if err := st.Seed(); err != nil {
		return nil, fmt.Errorf("seed store: %w", err)
	}

	c := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)

	rebalanceEvery, err := time.ParseDuration(cfg.RebalanceEvery)
	if err != nil {
		return nil, fmt.Errorf("parse rebalance interval: %w", err)
	}

	engine := service.NewEngine(st, c, rebalanceEvery)
	sealer := service.NewCredentialSealer(cfg.ConnectionSecret)
	connectionService := service.NewPlatformConnectionService(
		st,
		sealer,
		service.NewInstagramConnector(cfg.MetaGraphBaseURL, cfg.MetaGraphAPIVersion),
	)
	metaOAuth := service.NewMetaOAuthService(
		cfg.MetaAppID,
		cfg.MetaAppSecret,
		cfg.MetaRedirectURI,
		cfg.MetaGraphBaseURL,
		cfg.MetaGraphAPIVersion,
		cfg.MetaOAuthScopes,
	)
	handler := httpapi.NewHandler(engine, connectionService, metaOAuth)

	return &App{
		store:             st,
		cache:             c,
		engine:            engine,
		connectionService: connectionService,
		metaOAuth:         metaOAuth,
		handler:           handler,
	}, nil
}

func (a *App) RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api/v1")
	a.handler.Register(api)
}

func (a *App) Start(ctx context.Context) {
	a.engine.Start(ctx)
}

func (a *App) Close() error {
	return a.cache.Close()
}
