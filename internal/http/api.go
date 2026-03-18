package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/ilovelili/ad-engine/internal/service"
)

type Handler struct {
	engine *service.Engine
}

func NewHandler(engine *service.Engine) *Handler {
	return &Handler{engine: engine}
}

func (h *Handler) Register(g *echo.Group) {
	g.GET("/healthz", h.Health)
	g.GET("/dashboard", h.Dashboard)
	g.POST("/rebalance", h.Rebalance)
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (h *Handler) Dashboard(c echo.Context) error {
	snapshot, err := h.engine.Dashboard()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if snapshot == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no active campaign"})
	}
	return c.JSON(http.StatusOK, snapshot)
}

func (h *Handler) Rebalance(c echo.Context) error {
	if err := h.engine.RunCycle(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	snapshot, err := h.engine.Dashboard()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, snapshot)
}
