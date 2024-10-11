package http

import (
	"github.com/gin-gonic/gin"
)

func NewRouter(repos *Repository) *gin.Engine {
	router := gin.New()
	router.POST("/api/user/register", repos.Register)
	router.POST("/api/user/login", repos.Authentication)
	router.POST("api/user/orders", AuthMiddleware(), repos.Loading)
	router.GET("/api/user/orders", AuthMiddleware(), repos.GetOrders)
	return router
}
