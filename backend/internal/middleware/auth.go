package middleware

import (
	"net/http"
	"strings"

	"github.com/Takapon2512/recipe-recommend/backend/internal/cognito"
	"github.com/Takapon2512/recipe-recommend/backend/internal/config"
	"github.com/gin-gonic/gin"
)

const ContextKeyClaims = "claims"

func Auth(cfg *config.Config) gin.HandlerFunc {
	verifier := cognito.NewVerifier(cfg)

	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or missing token"},
			})
			return
		}

		claims, err := verifier.Verify(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or missing token"},
			})
			return
		}

		c.Set(ContextKeyClaims, claims)
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}
