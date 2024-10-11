package http

import (
	"net/http"

	"io"
	"strconv"
	"time"

	json2 "github.com/A1extop/loyalty/internal/json"
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

func (r *Repository) Register(c *gin.Context) { //// После регистрации сделать так, что ты автоматом и авторизован
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
	user, err := json2.UnpackingJSON(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
	}
	r.Storage.AddUsers(user.Login, "secretKey", user.Password, c)

}

func validNumber(numberStr string) bool {
	var sum int
	alt := false
	for i := len(numberStr) - 1; i >= 0; i-- {
		num, err := strconv.Atoi(string(numberStr[i]))
		if err != nil {
			return false
		}
		if alt {
			num *= 2
			if num > 9 {
				num -= 9
			}
		}
		sum += num
		alt = !alt
	}
	return sum%10 == 0
}

func (r *Repository) GetOrders(c *gin.Context) {
	authToken, err := c.Cookie("auth_token")
	if err != nil {
		c.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}
	userName, err := jwt1.ParseJWT(authToken)
	if err != nil {
		c.String(http.StatusUnauthorized, "Invalid token.")
		return
	}
	history, err := r.Storage.Orders(userName, c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
	}
	if len(history) == 0 {
		c.Status(http.StatusNoContent) ////////////////////////////////////////////////
		return
	}
	data, err := json2.PackingHistoryJSON(history)
	_, err = c.Writer.Write(data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Data writing error")
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}
func (r *Repository) Loading(c *gin.Context) {
	if c.GetHeader("Content-Type") != "text/plain" {
		return
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading order number")
		return
	}
	numberString := string(data)
	ex := validNumber(numberString)
	if !ex {
		c.String(http.StatusUnprocessableEntity, "Invalid order number")
		return
	}
	userName, exists := c.Get("username")
	login := userName.(string)

	exists = r.Storage.CheckUserOrders(c, login, numberString)
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
	user, err := json2.UnpackingJSON(c)
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
