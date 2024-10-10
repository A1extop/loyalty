package http

import (
	"net/http"

	"strconv"
	"time"

	"github.com/A1extop/loyalty/internal/json"
	jwt1 "github.com/A1extop/loyalty/internal/jwt"
	"github.com/A1extop/loyalty/internal/store"
	"github.com/gin-gonic/gin"
)

type Repository struct {
	Storage store.Storage
}

func NewRepository(s store.Storage) *Repository {
	return &Repository{Storage: s}
}

func (r *Repository) Register(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
	user, err := json.UnpackingJSON(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
	}
	r.Storage.AddUsers(user.Login, "secretKey", user.Password, c)

}

func validNumber(numberStr string) bool {
	var sum int
	length := len(numberStr)
	for i := length; i > 0; i-- {
		num, err := strconv.Atoi(string(numberStr[i]))
		if err != nil {
			return false
		}
		if i%2 != 0 {
			num *= 2
		}
		sum += num
	}
	if sum%10 != 0 {
		return false
	}
	return true
}

func (r *Repository) Loading(c *gin.Context) {
	if c.GetHeader("Content-Type") != "text/plain" {
		return
	}
	data := make([]byte, 35) // достаточно ли?
	reader := c.Request.Body
	n, err := reader.Read(data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading order number")
		return
	}
	data = data[:n] // добавить проверку на корректность введенных данных (Формат Луна)

	numberString := string(data)
	ex := validNumber(numberString)
	if !ex {
		c.String(http.StatusUnauthorized, "Not valid number") // не уверен, что верно
		return
	}
	userName, exists := c.Get("username")
	login := userName.(string)

	exists = r.Storage.CheckUserOrders(c, login)
	if exists {
		return
	}
	ok := r.Storage.CheckNumber(c, numberString)
	if !ok {
		return
	}
	r.Storage.SendingData(login, numberString, c)
}
func (r *Repository) Authentication(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
	user, err := json.UnpackingJSON(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
	}
	exists := r.Storage.CheckAvailability(user.Login, user.Password, c)
	token, err := jwt1.GenerateJWT(user.Login)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token")
		return
	}
	if exists {
		cookie := &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(c.Writer, cookie)

		c.String(http.StatusOK, "user successfully authenticated")

	}

}
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		cookie, err := c.Cookie("auth_token")
		if err != nil {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		claims, err := jwt1.ValidateToken(cookie)
		if err != nil {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}
