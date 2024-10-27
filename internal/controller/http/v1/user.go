package v1

import (
	"net/http"

	"github.com/A1extop/loyalty/internal/domain"
	jwt1 "github.com/A1extop/loyalty/internal/jwt"
	"github.com/A1extop/loyalty/internal/services/users/interfaces"
	"github.com/A1extop/loyalty/internal/services/users/models"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service interfaces.IUserCase
}

func NewUserHandler(engine *gin.Engine, service interfaces.IUserCase) { // checker middleware.IChecker
	handler := &UserHandler{
		service: service,
	}
	// делить
	router := engine.Group("/api")
	{
		//router.POST("/user/register", checker.AuthorizeRoles(util.Admin), handler.Register)   а checker.AuthorizeRoles(util.Admin)
		router.POST("/user/register", handler.Register)
		router.POST("/user/login", handler.Authentication)

	}
}

func (h *UserHandler) Register(ctx *gin.Context) {
	var user models.UserCredentials
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error1": err.Error()})
		return
	}
	err := h.service.AddAccount(ctx, &user)
	if err != nil {
		ctx.JSON(domain.StatusDetermination(err), gin.H{
			"error2": err.Error(),
		})
		return
	}

	err = jwt1.GenerateJWT(ctx, user.Login)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error3": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "User created successfully!",
	})
}

func (h *UserHandler) Authentication(ctx *gin.Context) {
	var user models.UserCredentials
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.service.AuthenticationAccount(ctx, &user)
	if err != nil {
		ctx.JSON(domain.StatusDetermination(err), gin.H{"error": err.Error()})
		return
	}
	err = jwt1.GenerateJWT(ctx, user.Login)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{

		"message": "User authentication",
	})
}
