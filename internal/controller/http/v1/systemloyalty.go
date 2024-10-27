package v1

import (
	"context"
	"time"

	"github.com/A1extop/loyalty/internal/services/systemloyalty/interfaces"
)

type SystemLoyaltyHandler struct {
	service interfaces.ISystemLoyaltyCase
}

func Action(ctx context.Context, service interfaces.ISystemLoyaltyCase, ticker *time.Ticker, systemAddr string) {
	handler := &SystemLoyaltyHandler{
		service: service,
	}
	go handler.service.InteractionWithCalculationSystem(ctx, ticker, systemAddr)
}
