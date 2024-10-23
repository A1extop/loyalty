package http

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"time"

	domain "github.com/A1extop/loyalty/internal/domain"
	json2 "github.com/A1extop/loyalty/internal/json"
	jwt1 "github.com/A1extop/loyalty/internal/jwt"
	"github.com/A1extop/loyalty/internal/store"
	"github.com/A1extop/loyalty/internal/usecase"
	"github.com/gin-gonic/gin"
)

type Repository struct {
	Storage store.Storage
}

func NewRepository(s store.Storage) *Repository {
	return &Repository{Storage: s}
}
func setAuthCookie(c *gin.Context, name string, value string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(c.Writer, cookie)
}

// Регистрирует пользователя и после успешной регистрации сразу авторизовывает
func (r *Repository) Register(c *gin.Context) {
	data := c.Request.Body
	user, err := json2.UnpackingUserJSON(data)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	status, err := usecase.AddAccount(r.Storage, user)
	if err != nil {
		c.String(status, err.Error())
		return
	}

	token, err := jwt1.GenerateJWT(user.Login)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	setAuthCookie(c, "auth_token", token)
	c.String(http.StatusOK, "user successfully registered")
}

// Обработка заказов
func processOrder(r *Repository, order string, systemAddr string) error {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	url := fmt.Sprintf("%s/api/orders/%s", systemAddr, order)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var orderResp *json2.OrderResponse2
	if resp.StatusCode == http.StatusOK && resp.Header.Get("Content-Type") == "application/json" {
		orderResp, err = json2.UnpackingSystemResponse(resp.Body)
		if err != nil {
			return err
		}
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		time.Sleep(60 * time.Second)
	}
	if orderResp == nil || orderResp.Accrual == nil {
		return nil
	}
	sum := *orderResp.Accrual
	err = r.Storage.UpdateOrderInDB(order, orderResp.Status, int(sum*100))
	if err != nil {
		return err
	}

	return nil
}

// Взаимодействие с системой расчёта начислений
func (r *Repository) InteractionWithCalculationSystem(ticker *time.Ticker, systemAddr string) {
	for range ticker.C {
		orders, err := r.Storage.GetOrdersForProcessing()
		if err != nil {
			log.Printf("Error fetching orders for processing: %v\n", err)
			continue
		}

		for _, order := range orders {
			err := processOrder(r, order, systemAddr)
			if err != nil {
				log.Printf("Error processing order %s: %v\n", order, err)
			}
		}
	}
}

// Получение информации о выводе средств
func (r *Repository) GetWithdrawals(c *gin.Context) {
	userName, exists := c.Get("username")
	if !exists {
		c.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}
	status, dataByte, err := usecase.GetWithdrawals(r.Storage, userName.(string))
	if err != nil {
		c.String(status, err.Error())
	}
	c.Data(status, "application/json", dataByte)

}

// Получение заказов
func (r *Repository) GetOrders(c *gin.Context) {
	userName, exists := c.Get("username")
	if !exists {
		c.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}
	status, data, err := usecase.GetOrders(r.Storage, userName.(string))
	if err != nil {
		c.String(status, err.Error())
		return
	}
	c.Data(status, "application/json", data)
}

// Списание баллов
func (r *Repository) PointsDebiting(c *gin.Context) {

	userName, exists := c.Get("username")
	if !exists {
		c.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}

	data := c.Request.Body
	orderPoints, err := json2.UnpackingOrderPointsJSON(data)
	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	status, err := usecase.WriteOff(r.Storage, userName.(string), orderPoints.Order, orderPoints.Sum)
	if err != nil {
		c.String(status, err.Error())
		return
	}
	c.String(status, "successful write-off")
}

// Получение баланса пользователем
func (r *Repository) GetBalance(c *gin.Context) {
	userName, exists := c.Get("username")
	if !exists {
		c.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}
	status, data, err := usecase.GetBalance(r.Storage, userName.(string))
	if err != nil {
		c.String(status, err.Error())
		return
	}
	c.Data(status, "application/json", data)
}

// Загрузка номера заказа
func (r *Repository) Loading(c *gin.Context) {

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading order number")
		return
	}
	numberString := string(body)

	userName, exists := c.Get("username")
	if !exists {
		c.String(http.StatusUnauthorized, "User is not authenticated")
		return
	}
	login := userName.(string)

	status, err := usecase.Load(r.Storage, numberString, login)
	if err != nil {
		c.String(status, err.Error())
		return
	}
	c.Status(status)
}

// Аутенфикация пользователя
func (r *Repository) Authentication(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		c.Status(http.StatusBadRequest)
		return
	}
	data := c.Request.Body
	user, err := json2.UnpackingUserJSON(data)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	status, err := usecase.AuthenticationAccount(r.Storage, user)
	if err != nil {
		c.String(status, err.Error())
		return
	}

	token, err := jwt1.GenerateJWT(user.Login)
	if err != nil {
		c.String(domain.StatusDetermination(err), err.Error())
		return
	}
	setAuthCookie(c, "auth_token", token)

	c.String(status, "user successfully authenticated")

}
