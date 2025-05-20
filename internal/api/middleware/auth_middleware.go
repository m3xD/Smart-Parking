package middleware

import (
	"log"
	"net/http"
	"smart_parking/internal/service" // Import AuthService
	"strings"

	"github.com/gin-gonic/gin"
	// "github.com/golang-jwt/jwt/v5" // Không cần trực tiếp ở đây nếu service xử lý
)

const (
	AuthorizationHeaderKey  = "Authorization"
	AuthorizationTypeBearer = "Bearer"
	UserIDKey               = "userID"
	UserRoleKey             = "userRole"
	UsernameKey             = "username"
)

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

// Authenticate là middleware để xác thực JWT
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Thiếu authorization header"})
			return
		}

		fields := strings.Fields(authHeader)
		if len(fields) < 2 || !strings.EqualFold(fields[0], AuthorizationTypeBearer) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Định dạng authorization header không hợp lệ"})
			return
		}

		accessToken := fields[1]
		_, claims, err := m.authService.ValidateToken(accessToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc đã hết hạn", "details": err.Error()})
			return
		}

		// Lấy userID và role từ claims (đảm bảo chúng tồn tại và đúng kiểu)
		userIDStr, okUserID := claims["sub"].(string) // Subject thường là ID
		userRole, okUserRole := claims["role"].(string)
		username, okUsername := claims["username"].(string)

		if !okUserID || !okUserRole || !okUsername {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Thông tin người dùng trong token không hợp lệ"})
			return
		}

		// Lưu thông tin người dùng vào context của Gin để các handler sau có thể sử dụng
		c.Set(UserIDKey, userIDStr) // Có thể cần parse thành int nếu cần
		c.Set(UserRoleKey, userRole)
		c.Set(UsernameKey, username)

		c.Next() // Tiếp tục xử lý request
	}
}

// AuthorizeRole là middleware để kiểm tra vai trò
func (m *AuthMiddleware) AuthorizeRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleVal, exists := c.Get(UserRoleKey)
		if !exists {
			log.Printf("AuthorizeRole: Không tìm thấy vai trò người dùng trong context (cần Authenticate() trước)")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Không có quyền truy cập (thiếu vai trò)"})
			return
		}

		userRole, ok := userRoleVal.(string)
		if !ok {
			log.Printf("AuthorizeRole: Định dạng vai trò người dùng không hợp lệ trong context")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Không có quyền truy cập (vai trò không hợp lệ)"})
			return
		}

		authorized := false
		for _, reqRole := range requiredRoles {
			if userRole == reqRole {
				authorized = true
				break
			}
		}

		if !authorized {
			log.Printf("AuthorizeRole: Người dùng với vai trò '%s' không có quyền truy cập (yêu cầu: %v)", userRole, requiredRoles)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Không có quyền truy cập (vai trò không phù hợp)"})
			return
		}

		log.Printf("AuthorizeRole: Người dùng với vai trò '%s' được phép truy cập (yêu cầu: %v)", userRole, requiredRoles)
		c.Next()
	}
}
