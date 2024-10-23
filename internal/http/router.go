package http

import (
	"github.com/A1extop/loyalty/internal/compress"
	"github.com/A1extop/loyalty/internal/logging"
	"github.com/gin-gonic/gin"
)

type ReposHandler struct {
	Repos Repository
}

func NewRouter(repos *Repository) *gin.Engine {
	router := gin.New()
	log := logging.New()
	router.POST("/api/user/register", compress.CompressData(), logging.LoggingPost(log), repos.Register)
	router.POST("/api/user/login", compress.DeCompressData(), logging.LoggingPost(log), repos.Authentication)
	router.POST("api/user/orders", compress.CompressData(), logging.AuthMiddleware(), logging.LoggingPost(log), repos.Loading)
	router.GET("/api/user/orders", compress.DeCompressData(), logging.AuthMiddleware(), logging.LoggingGet(log), repos.GetOrders)
	router.GET("/api/user/balance", compress.DeCompressData(), logging.AuthMiddleware(), logging.LoggingGet(log), repos.GetBalance)
	router.POST("/api/user/balance/withdraw", compress.CompressData(), logging.AuthMiddleware(), logging.LoggingPost(log), repos.PointsDebiting)
	router.GET("/api/user/withdrawals", compress.DeCompressData(), logging.AuthMiddleware(), logging.LoggingGet(log), repos.GetWithdrawals)

	return router
}
