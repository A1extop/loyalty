package json

import (
	"encoding/json"
	"time"

	"io"
)

type UserCredentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type History struct {
	Order       string    `json:"order"`
	Username    string    `json:"username"`
	Status      string    `json:"status"`
	Accrual     float64   `json:"accrual,omitempty"`
	Withdrawals float64   `json:"withdrawals"`
	Uploaded    time.Time `json:"uploaded_at"`
}
type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type OrderPoints struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type OrderResponse struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}
type OrderResponse1 struct {
	Order    string    `json:"number"`
	Status   string    `json:"status"`
	Accrual  float64   `json:"accrual,omitempty"`
	Uploaded time.Time `json:"uploaded_at"`
}
type OrderResponse2 struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

func NewOrderPoints() *OrderPoints {
	return &OrderPoints{}
}
func NewUser() *UserCredentials {
	return &UserCredentials{}
}

func PackingWithdrawalsJSON(slHistory []History) ([]byte, error) {
	type PartialHistory struct {
		Order       string    `json:"order"`
		Withdrawals float64   `json:"sum"`
		Uploaded    time.Time `json:"processed_at"`
	}
	partialHistorys := make([]PartialHistory, 0)
	for i := range slHistory {
		partial := PartialHistory{
			Order:       slHistory[i].Order,
			Withdrawals: slHistory[i].Withdrawals,
			Uploaded:    slHistory[i].Uploaded,
		}
		partialHistorys = append(partialHistorys, partial)
	}
	data, err := json.Marshal(partialHistorys)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Возвращает структуру с полями current и withdrawn
func NewBalance(current float64, withdrawn float64) *Balance {
	return &Balance{Current: current, Withdrawn: withdrawn}
}

// Распаковывает данные в структуру OrderPoints
func UnpackingOrderPointsJSON(data io.ReadCloser) (*OrderPoints, error) {
	orderPoints := NewOrderPoints()
	if err := json.NewDecoder(data).Decode(orderPoints); err != nil {
		return nil, err
	}
	return orderPoints, nil
}

// Распаковывает данные в структуру UserCredentials и возвращает структуру
func UnpackingUserJSON(data io.ReadCloser) (*UserCredentials, error) {
	user := NewUser()
	if err := json.NewDecoder(data).Decode(user); err != nil {
		return nil, err
	}
	return user, nil
}

// Заворачивание данных в JSON
func PackingHistoryJSON(slHistory []History) ([]byte, error) {
	responses := make([]OrderResponse1, len(slHistory))

	for i := range slHistory {
		slHistory[i].Uploaded = slHistory[i].Uploaded.UTC()
		responses[i] = OrderResponse1{
			Order:    slHistory[i].Order,
			Status:   slHistory[i].Status,
			Accrual:  slHistory[i].Accrual,
			Uploaded: slHistory[i].Uploaded,
		}
	}

	data, err := json.Marshal(responses)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Упаковывает структуру Balance в формат JSON
func PackingMoney(balance Balance) ([]byte, error) {
	data, err := json.Marshal(balance)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func UnpackingOrderResponseJSON(body []byte) (*OrderResponse, error) {
	var orderResponse *OrderResponse
	if err := json.Unmarshal(body, &orderResponse); err != nil {
		return nil, err
	}
	return orderResponse, nil
}

func UnpackingSystemResponse(data io.ReadCloser) (*OrderResponse2, error) {
	var orderResponse *OrderResponse2
	if err := json.NewDecoder(data).Decode(orderResponse); err != nil {
		return nil, err
	}
	return orderResponse, nil
}
