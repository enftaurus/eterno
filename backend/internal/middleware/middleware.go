package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eterno/backend/internal/auth"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextKeyClaims        = "claims"
	ContextKeyUserID        = "user_id"
	ContextKeyRole          = "role"
	ContextKeyInstitutionID = "institution_id"
	ContextKeyRequestID     = "request_id"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Set(ContextKeyRequestID, reqID)
		c.Header("X-Request-ID", reqID)
		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		reqID, _ := c.Get(ContextKeyRequestID)

		if raw != "" {
			path = path + "?" + raw
		}

		slog.Info("HTTP Request",
			"status", status,
			"method", method,
			"path", path,
			"ip", clientIP,
			"latency_ms", latency.Milliseconds(),
			"request_id", reqID,
		)
	}
}

func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allow := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allow = true
				break
			}
		}

		if allow && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Length, X-CSRF-Token, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization format, expected Bearer <token>"})
			return
		}

		claims, err := auth.ValidateToken(cfg, parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyInstitutionID, claims.InstitutionID)

		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Malformed role context"})
			return
		}

		if roleStr == models.RolePlatformAdmin {
			c.Next()
			return
		}

		for _, r := range roles {
			if r == roleStr {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for this resource"})
	}
}

func EnsureTenantScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(ContextKeyRole)
		if role == models.RolePlatformAdmin {
			c.Next()
			return
		}

		userInstID, _ := c.Get(ContextKeyInstitutionID)
		instParam := c.Param("institution_id")
		if instParam != "" && userInstID != "" && instParam != userInstID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Cross-tenant data access strictly prohibited"})
			return
		}

		c.Next()
	}
}
