package v1

import (
	"net/http"

	"github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/logging"
	"github.com/A1extop/loyalty/internal/services/loyalty/interfaces"
	"github.com/A1extop/loyalty/internal/services/loyalty/models"
	"github.com/gin-gonic/gin"
)

type LoyaltyHandler struct {
	service interfaces.ILoyaltyCase
}

func NewLoyaltyHandler(engine *gin.Engine, service interfaces.ILoyaltyCase) { // checker middleware.IChecker
	handler := &LoyaltyHandler{
		service: service,
	}
	// делить
	router := engine.Group("/api")
	{
		router.GET("/user/balance", logging.AuthMiddleware(), handler.GetBalance)
		router.POST("/user/balance/withdraw", logging.AuthMiddleware(), handler.PointsDebiting)
	}
}
func (h *LoyaltyHandler) GetBalance(ctx *gin.Context) {
	userName, exists := ctx.Get("username") ////вопрос
	if !exists {
		ctx.String(http.StatusUnauthorized, "The user is not authorized.")
		return
	}

	current, withdrawn, err := h.service.GetBalanceAccount(ctx, userName.(string))
	if err != nil {
		ctx.JSON(domain.StatusDetermination(err), gin.H{"error": err.Error()})
		return
	}
	balance := models.Balance{Current: current, Withdrawn: withdrawn}
	ctx.JSON(http.StatusOK, balance)
}
func (h *LoyaltyHandler) PointsDebiting(ctx *gin.Context) {
	userName, exists := ctx.Get("username")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User is not authenticated"})
		return

	}

	var orderPoints models.OrderPoints
	if err := ctx.ShouldBindJSON(&orderPoints); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.service.WriteOff(ctx, userName.(string), orderPoints.Order, orderPoints.Sum)
	if err != nil {
		ctx.JSON(domain.StatusDetermination(err), gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Successful write-off of points",
	})
}
