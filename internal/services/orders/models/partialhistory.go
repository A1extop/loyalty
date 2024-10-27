package models

import (
	"time"
)

type PartialHistory struct {
	Order       string    `json:"order"`
	Withdrawals float64   `json:"sum"`
	Uploaded    time.Time `json:"processed_at"`
}
