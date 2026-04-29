package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetHeader("X-User-Id")
		userNameHeader := c.GetHeader("X-User-Name")
		if userId == "" || userNameHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing user headers",
			})
			return
		}

		userName, err := url.QueryUnescape(userNameHeader)
		if err != nil || userName == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user headers",
			})
			return
		}

		c.Set("userId", userId)
		c.Set("userName", userName)
		// "member" | "guest"
		c.Set("userKind", c.GetHeader("X-User-Kind"))
		c.Next()
	}
}
