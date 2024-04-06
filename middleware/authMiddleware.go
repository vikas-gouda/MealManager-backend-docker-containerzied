package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vikas-gouda/go-restraunt-mangement/helpers"
)

func Authentication() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Retrieve JWT token from the Authorization header.
		clientToken := ctx.Request.Header.Get("token")
		if clientToken == "" {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "No Authorization header Provided"})
			ctx.Abort()
			return
		}

		// Validate the JWT token.
		claims, err := helpers.ValidateToken(clientToken)
		if err != "" {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
			ctx.Abort()
			return
		}

		// Set user claims as context values for further processing.
		ctx.Set("email", claims.Email)
		ctx.Set("first_name", claims.First_name)
		ctx.Set("last_name", claims.Last_name)
		ctx.Set("uid", claims.Uid)
		// Proceed to the next middleware or handler.
		ctx.Next()
	}
}
