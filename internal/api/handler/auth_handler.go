package handler

import (
	"errors" // Thêm import
	"net/http"
	"smart_parking/internal/domain"
	"smart_parking/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(as *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: as}
}

// POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var dto domain.RegisterUserDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đăng ký người dùng", "details": err.Error()})
		return
	}
	// Không trả về password
	user.Password = ""
	c.JSON(http.StatusCreated, user)
}

// POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var dto domain.LoginUserDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResponse, err := h.authService.Login(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đăng nhập", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, authResponse)
}
