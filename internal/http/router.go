package http

import (
	"github.com/A1extop/loyalty/internal/logging"
	"github.com/gin-gonic/gin"
)

func NewRouter(repos *Repository) *gin.Engine {
	router := gin.New()
	log := logging.New()
	router.POST("/api/user/register", logging.LoggingPost(log), repos.Register)                          //+
	router.POST("/api/user/login", logging.LoggingPost(log), repos.Authentication)                       //+
	router.POST("api/user/orders", logging.AuthMiddleware(), logging.LoggingPost(log), repos.Loading)    // +
	router.GET("/api/user/orders", logging.AuthMiddleware(), logging.LoggingGet(log), repos.GetOrders)   // +
	router.GET("/api/user/balance", logging.AuthMiddleware(), logging.LoggingGet(log), repos.GetBalance) //+
	router.POST("/api/user/balance/withdraw", logging.AuthMiddleware(), logging.LoggingPost(log), repos.PointsDebiting)
	router.GET("/api/user/withdrawals", logging.AuthMiddleware(), logging.LoggingGet(log), repos.GetWithdrawals)

	return router
}
