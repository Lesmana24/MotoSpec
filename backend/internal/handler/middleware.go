package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/motospec/backend/internal/llm"
)

const (
	HeaderUserID         = "X-User-ID"
	HeaderSimulateOutage = "X-Simulate-Outage"
	DefaultUserID        = "demo-user"
	ContextKeyUserID     = "userID"
)

// UserIDMiddleware extracts X-User-ID header or defaults to "demo-user".
func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := strings.TrimSpace(c.GetHeader(HeaderUserID))
		if userID == "" {
			userID = DefaultUserID
		}
		c.Set(ContextKeyUserID, userID)
		c.Next()
	}
}

// OutageSimulationMiddleware inspects X-Simulate-Outage and injects the context hook.
func OutageSimulationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		outageHeader := strings.TrimSpace(strings.ToLower(c.GetHeader(HeaderSimulateOutage)))
		if outageHeader == "primary" {
			ctx := llm.WithSimulatedPrimaryOutage(c.Request.Context())
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}

// CORSMiddleware configures standard CORS headers for PWA clients.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-User-ID, X-Simulate-Outage")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// GetUserID retrieves the scoped user ID from Gin context.
func GetUserID(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyUserID); exists {
		if id, ok := val.(string); ok && id != "" {
			return id
		}
	}
	return DefaultUserID
}
