package httpapi

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/ilovelili/ad-engine/internal/domain"
	"github.com/ilovelili/ad-engine/internal/service"
)

const metaOAuthStateCookie = "meta_oauth_state"

type Handler struct {
	engine            *service.Engine
	connectionService *service.PlatformConnectionService
	metaOAuth         *service.MetaOAuthService
}

func NewHandler(engine *service.Engine, connectionService *service.PlatformConnectionService, metaOAuth *service.MetaOAuthService) *Handler {
	return &Handler{
		engine:            engine,
		connectionService: connectionService,
		metaOAuth:         metaOAuth,
	}
}

func (h *Handler) Register(g *echo.Group) {
	g.GET("/healthz", h.Health)
	g.GET("/dashboard", h.Dashboard)
	g.POST("/rebalance", h.Rebalance)
	g.GET("/connections", h.ListConnections)
	g.POST("/connections", h.ConnectPlatform)
	g.GET("/oauth/meta/start", h.StartMetaOAuth)
	g.GET("/oauth/meta/callback", h.MetaOAuthCallback)
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

func (h *Handler) ListConnections(c echo.Context) error {
	view, err := h.connectionService.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, view)
}

func (h *Handler) ConnectPlatform(c echo.Context) error {
	var req service.ConnectPlatformRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
	}

	snapshot, err := h.connectionService.Connect(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, snapshot)
}

func (h *Handler) StartMetaOAuth(c echo.Context) error {
	state, err := service.GenerateOAuthState()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	authorizeURL, err := h.metaOAuth.AuthorizeURL(state)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	c.SetCookie(&http.Cookie{
		Name:     metaOAuthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((10 * time.Minute).Seconds()),
	})

	return c.Redirect(http.StatusTemporaryRedirect, authorizeURL)
}

func (h *Handler) MetaOAuthCallback(c echo.Context) error {
	if errValue := c.QueryParam("error"); errValue != "" {
		return c.Redirect(http.StatusTemporaryRedirect, h.homeRedirect("oauth_error", firstNonEmptyString(c.QueryParam("error_description"), errValue)))
	}

	code := c.QueryParam("code")
	state := c.QueryParam("state")
	if code == "" || state == "" {
		return c.Redirect(http.StatusTemporaryRedirect, h.homeRedirect("oauth_error", "Meta OAuth callback is missing code or state"))
	}

	stateCookie, err := c.Cookie(metaOAuthStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != state {
		return c.Redirect(http.StatusTemporaryRedirect, h.homeRedirect("oauth_error", "Meta OAuth state validation failed"))
	}

	c.SetCookie(&http.Cookie{
		Name:     metaOAuthStateCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	token, err := h.metaOAuth.ExchangeCode(c.Request().Context(), code)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, h.homeRedirect("oauth_error", err.Error()))
	}

	snapshot, err := h.connectionService.Connect(c.Request().Context(), service.ConnectPlatformRequest{
		Platform: domain.PlatformInstagram,
		Secret:   token,
	})
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, h.homeRedirect("oauth_error", err.Error()))
	}

	message := fmt.Sprintf("Connected %s and synced %d ad account(s)", firstNonEmptyString(snapshot.DisplayName, snapshot.AccountIdentifier), len(snapshot.AdAccounts))
	return c.Redirect(http.StatusTemporaryRedirect, h.homeRedirect("oauth_success", message))
}

func (h *Handler) homeRedirect(status, message string) string {
	query := url.Values{
		"oauth": []string{status},
	}
	if message != "" {
		query.Set("message", message)
	}
	return "/?" + query.Encode()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
