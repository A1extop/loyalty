package usecase

import (
	"context"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/A1extop/loyalty/internal/services/systemloyalty/interfaces"
	"github.com/A1extop/loyalty/internal/services/systemloyalty/models"
)

type SystemLoyaltyUsecase struct {
	repo interfaces.ISystemLoyaltyRepository
}

func NewSystemLoyaltyUsecase(repo interfaces.ISystemLoyaltyRepository) interfaces.ISystemLoyaltyCase {
	return &SystemLoyaltyUsecase{repo: repo}
}

func (s *SystemLoyaltyUsecase) InteractionWithCalculationSystem(ctx context.Context, ticker *time.Ticker, systemAddr string) {
	for range ticker.C {
		orders, err := s.repo.GetOrdersForProcessing(ctx)
		if err != nil {
			log.Printf("Error fetching orders for processing: %v\n", err)
			continue
		}

		for _, order := range orders {
			err := s.ProcessOrder(ctx, order, systemAddr)
			if err != nil {
				log.Printf("Error processing order %s: %v\n", order, err)
			}
		}
	}
}
func (s *SystemLoyaltyUsecase) ProcessOrder(ctx context.Context, order string, systemAddr string) error {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	url := fmt.Sprintf("%s/api/orders/%s", systemAddr, order)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var orderResponse *models.OrderResponse
	if resp.StatusCode == http.StatusOK && resp.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(resp.Body).Decode(&orderResponse); err != nil {
			return err
		}
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		time.Sleep(60 * time.Second)
	}
	if orderResponse == nil || orderResponse.Accrual == nil {
		return nil
	}
	sum := *orderResponse.Accrual
	err = s.repo.UpdateOrderInDB(ctx, order, orderResponse.Status, int(sum*100))
	if err != nil {
		return err
	}

	return nil
}
