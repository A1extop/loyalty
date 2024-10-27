package internal

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/A1extop/loyalty/config"
	"github.com/gin-gonic/gin"
)

func Run(ctx context.Context, config *config.Config, router *gin.Engine) { // logger *zap.Logger
	notifContext, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  60 * time.Second,
		//Addr:         config.App.Host + ":" + config.App.Port, // todo uncomment
		Handler: router,
	}

	//logger.Info("listen: " + config.App.Host + ":" + config.App.Port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen: " + err.Error())
		}
	}()

	<-notifContext.Done()
	stop()
	log.Println("Shutting down graceful")

	notifContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(notifContext); err != nil {
		log.Fatal("Server forced to shutdown: " + err.Error())
	}

	log.Println("Server exiting")
}
