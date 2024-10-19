package http

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	errors2 "github.com/A1extop/loyalty/internal/errors"
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
func SetAuthCookie(c *gin.Context, name string, value string, duration time.Duration) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(duration),
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(c.Writer, cookie)
}

// Регистрирует пользователя и после успешной регистрации сразу авторизовывает
func (r *Repository) Register(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
	data := c.Request.Body
	user, err := json2.UnpackingUserJSON(data)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
	}
	err = r.Storage.AddUsers(user.Login, "secretKey", user.Password)
	if err != nil {
		c.String(errors2.StatusDetermination(err), err.Error())
	}
	token, err := jwt1.GenerateJWT(user.Login)
	if err != nil {
		c.String(errors2.StatusDetermination(err), err.Error())
		return
	}
	time := 24 * time.Hour
	SetAuthCookie(c, "auth_token", token, time)
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
	if orderResp.Accrual == nil {
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
	data, err := r.Storage.Orders(userName.(string))
	if err != nil {
		c.Status(http.StatusInternalServerError)
	}
	dataByte, err := json2.PackingWithdrawalsJSON(data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Data conversion error")
		return
	}
	c.Data(http.StatusOK, "application/json", dataByte)

}

// Получение заказов
func (r *Repository) GetOrders(c *gin.Context) {
	userName, exists := c.Get("username")
	if !exists {
		c.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}
	history, err := r.Storage.Orders(userName.(string))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to retrieve orders.", err.Error())
		return
	}
	if len(history) == 0 {
		c.Header("Content-Type", "application/json")
		c.Status(http.StatusNoContent)
		return
	}
	data, err := json2.PackingHistoryJSON(history)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to pack orders into JSON.")

		return
	}
	c.Data(http.StatusOK, "application/json", data)
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
	err = r.Storage.ChangeLoyaltyPoints(userName.(string), orderPoints.Order, orderPoints.Sum)
	if err != nil {
		c.String(errors2.StatusDetermination(err), err.Error())
		return
	}
	c.Status(http.StatusOK)
}

// Проверка номера на алгоритм Луна
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

// Получение баланса пользователем
func (r *Repository) GetBalance(c *gin.Context) {
	userName, exists := c.Get("username")
	if !exists {
		c.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}
	current, withdrawn, err := r.Storage.Balance(userName.(string))
	if err != nil {
		c.String(http.StatusInternalServerError, "error receiving balance", err.Error())
		return
	}
	balance := json2.NewBalance(current, withdrawn)

	data, err := json2.PackingMoney(*balance)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed packing JSON")
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// Загрузка номера заказа
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
	if !exists {
		c.String(http.StatusUnauthorized, "User is not authenticated")
	}
	login := userName.(string)

	exists, err = r.Storage.CheckUserOrders(login, numberString)
	if err != nil {
		c.String(errors2.StatusDetermination(err), err.Error())
	}
	if exists {
		c.String(http.StatusOK, "Everything is fine")
		return
	}
	err = r.Storage.SendingData(login, numberString)
	if err != nil {
		c.String(errors2.StatusDetermination(err), err.Error())
		return
	}
	c.String(http.StatusAccepted, "The new order number has been accepted for processing;")
}

// Аутенфикация пользователя
func (r *Repository) Authentication(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
	data := c.Request.Body
	user, err := json2.UnpackingUserJSON(data)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
	}
	err = r.Storage.CheckAvailability(user.Login, user.Password)
	if err != nil {
		errors2.StatusDetermination(err)

	}
	token, err := jwt1.GenerateJWT(user.Login)
	if err != nil {
		c.String(errors2.StatusDetermination(err), err.Error())
		return
	}
	time := 24 * time.Hour
	SetAuthCookie(c, "auth_token", token, time)

	c.String(http.StatusOK, "user successfully authenticated")

}
