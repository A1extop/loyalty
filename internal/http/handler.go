package http

import (
	"net/http"

	"fmt"
	"io"
	"log"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/A1extop/loyalty/internal/domain"
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
		c.String(domain.StatusDetermination(err), err.Error())
	}
	token, err := jwt1.GenerateJWT(user.Login)
	if err != nil {
		c.String(domain.StatusDetermination(err), err.Error())
		return
	}
	time := 24 * time.Hour
	SetAuthCookie(c, "auth_token", token, time)
	c.String(http.StatusOK, "user successfully registered")
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
	}
	err = r.Storage.ChangeLoyaltyPoints(userName.(string), orderPoints.Order, orderPoints.Sum)
	if err != nil {
		c.String(domain.StatusDetermination(err), err.Error())
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
	current /= 10
	withdrawn /= 10
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
		c.String(domain.StatusDetermination(err), err.Error())
	}
	if exists {
		c.String(http.StatusOK, "Everything is fine")
		return
	}
	err = r.Storage.CheckNumber(numberString)
	if err != nil {
		c.String(domain.StatusDetermination(err), err.Error())
		return
	}
	err = r.Storage.SendingData(login, numberString)
	if err != nil {
		c.String(domain.StatusDetermination(err), err.Error())
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
		domain.StatusDetermination(err)

	}
	token, err := jwt1.GenerateJWT(user.Login)
	if err != nil {
		c.String(domain.StatusDetermination(err), err.Error())
		return
	}
	time := 24 * time.Hour
	SetAuthCookie(c, "auth_token", token, time)

	c.String(http.StatusOK, "user successfully authenticated")

}

// Получение данных о заказе, если статус 200, распаковка их и отправка в канал result. Возвращает статус заказа
func FetchOrder(orderNumber string, wgResults *sync.WaitGroup, resultsChan chan<- json2.OrderResponse, apiURL string) int {
	defer wgResults.Done()

	resp, err := http.Get(fmt.Sprintf(apiURL, orderNumber))
	if err != nil {
		return http.StatusInternalServerError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusInternalServerError
	}

	orderResponse, err := json2.UnpackingOrderResponseJSON(body)
	if err != nil {
		return http.StatusInternalServerError
	}
	resultsChan <- *orderResponse
	return http.StatusOK
}

// Идёт по номерам в канале orderNumbers. С этим номером идёт в чёрный ящик. Если статус 429, то преккращает работу на 1 минуту
func worker(orderNumbers chan string, wg *sync.WaitGroup, resultsChan chan<- json2.OrderResponse, apiURL string, stopChan chan struct{}) {
	for orderNumber := range orderNumbers {
		wg.Add(1)

		go func(orderNumber string) {
			defer wg.Done()

			for {
				select {
				case <-stopChan:
					time.Sleep(1 * time.Minute)
					continue
				default:
					status := FetchOrder(orderNumber, wg, resultsChan, apiURL)

					if status == http.StatusOK {
						return
					} else if status == http.StatusTooManyRequests {
						log.Printf("Received status 429 for order %s, signal about suspension", orderNumber)
						stopChan <- struct{}{}
					} else {
						log.Printf("Order processing error %s: %d", orderNumber, status)
						orderNumbers <- orderNumber
						return
					}
				}
			}
		}(orderNumber)
	}
}

// Обращается к "черному ящику" за данными о заказе и отправку их в БД
func (r *Repository) WorkingWithLoyaltyCalculationService(apiURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var wg sync.WaitGroup
		numCPUs := runtime.NumCPU()

		orderChan := make(chan string, 100)
		resultsChan := make(chan json2.OrderResponse)
		stopChan := make(chan struct{})

		for i := 0; i < numCPUs; i++ {
			go worker(orderChan, &wg, resultsChan, apiURL, stopChan)
		}

		go r.Storage.FetchAndUpdateOrderNumbers(orderChan)

		go func() {
			for result := range resultsChan {
				if result.Status == "REGISTERED" || result.Status == "PROCESSING" {
					orderChan <- result.Order
				} else {
					err := r.Storage.Send(result)
					if err != nil {
						log.Println("error sending DB")
					}
				}
			}
		}()

		wg.Wait()
		close(orderChan)
		close(resultsChan)
		close(stopChan)

		c.JSON(http.StatusOK, gin.H{"message": "Calculation service completed"})
	}
}
