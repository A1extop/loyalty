package jwt

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

var cookieStore = cookie.NewStore([]byte("secretKey"))
var jwtKey = []byte("secretKey")

func GenerateJWT(c *gin.Context, username string) error {
	claims := &jwt.StandardClaims{
		Subject:   username,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return err
	}

	session := sessions.Default(c)
	session.Set("jwt_token", tokenString)
	if err := session.Save(); err != nil {
		return err
	}

	return nil
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		tokenString := session.Get("jwt_token")

		if tokenString == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No token in session"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString.(string), func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("invalid signing method", jwt.ValidationErrorSignatureInvalid)
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("user", claims["sub"])
		}

		c.Next()
	}
}
