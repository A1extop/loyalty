package models

import (
	"time"
)

type History struct {
	Order       string    `json:"order"`
	Username    string    `json:"username"`
	Status      string    `json:"status"`
	Accrual     float64   `json:"accrual,omitempty"`
	Withdrawals float64   `json:"withdrawals"`
	Uploaded    time.Time `json:"uploaded_at"`
}
