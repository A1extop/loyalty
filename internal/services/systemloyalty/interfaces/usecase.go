package interfaces

import (
	"context"
	"time"
)

type ISystemLoyaltyCase interface {
	InteractionWithCalculationSystem(ctx context.Context, ticker *time.Ticker, systemAddr string)
	ProcessOrder(ctx context.Context, order string, systemAddr string) error
}
