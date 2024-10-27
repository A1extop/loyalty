package v1

import (
	"net/http"

	"github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/jwt"
	"github.com/A1extop/loyalty/internal/services/orders/interfaces"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	service interfaces.IOrderCase
}

func NewOrderHandler(engine *gin.Engine, service interfaces.IOrderCase) { // checker middleware.IChecker
	handler := &OrderHandler{
		service: service,
	}

	router := engine.Group("/api")
	router.Use(jwt.AuthMiddleware())
	{
		router.POST("/user/orders", handler.Loading)
		router.GET("/user/orders", handler.GetOrders)
		router.GET("/user/withdrawals", handler.GetWithdrawals)
	}
}

func (h *OrderHandler) GetOrders(ctx *gin.Context) {
	userName, exists := ctx.Get("username")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User is not authenticated"})
		return

	}
	data, err := h.service.GetOrders(ctx, userName.(string))
	if err != nil {
		ctx.JSON(domain.StatusDetermination(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, data)

}
func (h *OrderHandler) GetWithdrawals(ctx *gin.Context) {
	username, exists := ctx.Get("username")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User is not authenticated"})
		return

	}
	history, err := h.service.GetWithdrawals(ctx, username.(string))
	if err != nil {
		ctx.JSON(domain.StatusDetermination(err), gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, history)
}

func (h *OrderHandler) Loading(ctx *gin.Context) {

	userName, exists := ctx.Get("username")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User is not authenticated"})
		return

	}
	data, err := ctx.GetRawData()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	numberString := string(data)
	login := userName.(string)

	ex, err := h.service.Load(ctx, numberString, login)
	if err != nil {
		ctx.JSON(domain.StatusDetermination(err), gin.H{"error": err.Error()})
		return
	}
	if ex {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Order has already been uploaded.",
		})
	}
	ctx.JSON(http.StatusAccepted, gin.H{
		"message": "New order uploaded",
	})
}
