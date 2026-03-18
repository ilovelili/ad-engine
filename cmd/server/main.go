package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/ilovelili/ad-engine/internal/app"
	"github.com/ilovelili/ad-engine/internal/config"
)

//go:embed all:web
var webAssets embed.FS

func main() {
	cfg := config.Load()

	appDeps, err := app.New(cfg)
	if err != nil {
		log.Fatalf("initialize app: %v", err)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	appDeps.RegisterRoutes(e)

	webFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		log.Fatalf("load static assets: %v", err)
	}

	e.GET("/", echo.WrapHandler(http.FileServerFS(webFS)))
	e.GET("/*", echo.WrapHandler(http.FileServerFS(webFS)))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go appDeps.Start(ctx)

	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalf("start http server: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	if err := appDeps.Close(); err != nil {
		log.Printf("close app dependencies: %v", err)
	}

	os.Exit(0)
}
