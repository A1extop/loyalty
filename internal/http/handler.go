package http

import (
	"net/http"

	"errors"
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
func (r *Repository) Register(c *gin.Context) { //// После регистрации сделать так, что ты автоматом и авторизован
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
	user, err := json2.UnpackingUserJSON(c)
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

func (r *Repository) PointsDebiting(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
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
	orderPoints, err := json2.UnpackingOrderPointsJSON(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
	}
	err = r.Storage.ChangeLoyaltyPoints(userName, orderPoints.Order, orderPoints.Sum)
	if err != nil {
		c.String(domain.StatusDetermination(err), err.Error())
	}
	c.Status(http.StatusOK)
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
func (r *Repository) Authentication(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		return
	}
	user, err := json2.UnpackingUserJSON(c)
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

func FetchOrder(orderNumber string, wg *sync.WaitGroup, wgResults *sync.WaitGroup, resultsChan chan<- json2.OrderResponse, apiURL string) error {
	defer wg.Done()
	defer wgResults.Done()

	resp, err := http.Get(fmt.Sprintf(apiURL, orderNumber))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("Status mismatch")
	}

	body, _ := io.ReadAll(resp.Body)

	orderResponse, err := json2.UnpackingOrderResponseJSON(body)
	if err != nil {
		return nil ///////очень не уверен, нужно ли вообще юзать здесь ерроры
	}
	resultsChan <- *orderResponse
	return nil
}

func worker(orderNumbers <-chan string, wg *sync.WaitGroup, resultsChan chan<- json2.OrderResponse, apiURL string) {
	for orderNumber := range orderNumbers {
		wg.Add(1)

		go func(orderNumber string) {
			defer wg.Done()

			err := FetchOrder(orderNumber, wg, wg, resultsChan, apiURL)
			if err != nil {
				log.Printf("Ошибка при обработке заказа %s: %v", orderNumber, err) ///////////точно убрать
			}
		}(orderNumber)
	}
}

func (r *Repository) WorkingWithLoyaltyCalculationService(c *gin.Context, apiURL string) {
	var wg sync.WaitGroup
	numCPUs := runtime.NumCPU()

	orderChan := make(chan string, 100)
	resultsChan := make(chan json2.OrderResponse)

	for i := 0; i < numCPUs; i++ {
		go worker(orderChan, &wg, resultsChan, apiURL)
	}

	go r.Storage.FetchAndUpdateOrderNumbers(orderChan)

	go func() {
		for result := range resultsChan {

			if result.Status == "REGISTERED" || result.Status == "PROCESSING" {
				orderChan <- result.Order
			} else {
				err := r.Storage.Send(result)
				if err != nil {
					log.Println("error sending data to database")
				}
			}
		}
	}()

	wg.Wait()
	close(resultsChan)
}
