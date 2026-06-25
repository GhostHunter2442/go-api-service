package handler

import (
	"net/http"
	"time"

	"github.com/apidet/go-api-service/internal/dto"
	"github.com/apidet/go-api-service/internal/service"
	"github.com/apidet/go-api-service/pkg/apperror"
	"github.com/apidet/go-api-service/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/api/v1/auth"
)

type AuthHandler struct {
	svc          *service.AuthService
	accessTTL    time.Duration
	refreshTTL   time.Duration
	secureCookie bool // true ใน production (HTTPS)
}

func NewAuthHandler(svc *service.AuthService, accessTTL, refreshTTL time.Duration, secureCookie bool) *AuthHandler {
	return &AuthHandler{svc: svc, accessTTL: accessTTL, refreshTTL: refreshTTL, secureCookie: secureCookie}
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, token, int(h.refreshTTL.Seconds()),
		refreshCookiePath, "", h.secureCookie, true) // Secure, HttpOnly
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", h.secureCookie, true)
}

func (h *AuthHandler) writeTokens(c *gin.Context, pair *service.TokenPair) {
	h.setRefreshCookie(c, pair.RefreshToken)
	c.JSON(http.StatusOK, response.Success(dto.TokenResponse{
		AccessToken: pair.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(h.accessTTL.Seconds()),
	}))
}

// Login
//
//	@Summary	Login ด้วย phone_number + password
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dto.LoginRequest	true	"credentials"
//	@Success	200		{object}	response.Body
//	@Failure	401		{object}	response.Body
//	@Router		/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("phone_number and password are required"))
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), req.PhoneNumber, req.Password)
	if err != nil {
		c.Error(err)
		return
	}
	h.writeTokens(c, pair)
}

// Refresh
//
//	@Summary	ขอ access token ใหม่ด้วย refresh cookie
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	response.Body
//	@Failure	401	{object}	response.Body
//	@Router		/api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	token, err := c.Cookie(refreshCookieName)
	if err != nil || token == "" {
		c.Error(apperror.Unauthorized("missing refresh token"))
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), token)
	if err != nil {
		h.clearRefreshCookie(c)
		c.Error(err)
		return
	}
	h.writeTokens(c, pair)
}

// Logout
//
//	@Summary	Logout (เพิกถอน refresh ทั้งหมด)
//	@Tags		auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Body
//	@Router		/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	if token, _ := c.Cookie(refreshCookieName); token != "" {
		_ = h.svc.Logout(c.Request.Context(), token)
	}
	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, response.Success(gin.H{"status": "logged out"}))
}
