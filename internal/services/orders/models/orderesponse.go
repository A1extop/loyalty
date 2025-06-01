package models

import (
	"time"
)

type OrderResponse struct {
	Order    string    `json:"number"`
	Status   string    `json:"status"`
	Accrual  float64   `json:"accrual,omitempty"`
	Uploaded time.Time `json:"uploaded_at"`
}
