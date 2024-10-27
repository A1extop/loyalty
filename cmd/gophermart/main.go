package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/A1extop/loyalty/config"
	"github.com/A1extop/loyalty/internal"
	v1 "github.com/A1extop/loyalty/internal/controller/http/v1"
	"github.com/A1extop/loyalty/internal/db"
	loRepo "github.com/A1extop/loyalty/internal/services/loyalty/repostory"
	loUse "github.com/A1extop/loyalty/internal/services/loyalty/usecase"
	orRepo "github.com/A1extop/loyalty/internal/services/orders/repostory"
	orUse "github.com/A1extop/loyalty/internal/services/orders/usecase"
	usRepo "github.com/A1extop/loyalty/internal/services/users/repostory"
	usUse "github.com/A1extop/loyalty/internal/services/users/usecase"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.New()
	cfg.Get()

	database, err := Init(ctx, cfg.AddrDB)
	if err != nil {
		log.Fatalln("Failed to connect to database at startup:", err)
	}
	db.CreateTable(ctx, database)
	userRepo := usRepo.NewUserRepo(database)
	userUsecase := usUse.NewUserUsecase(userRepo)

	orderRepo := orRepo.NewOrderRepo(database)
	orderUsecase := orUse.NewOrderUsecase(orderRepo)

	loyaltyRepo := loRepo.NewLoyaltyRepo(database)
	loyaltyUsecase := loUse.NewLoyaltyUsecase(loyaltyRepo)

	router := gin.Default()
	router.Use(cors.Default())
	v1.NewUserHandler(router, userUsecase)

	v1.NewOrderHandler(router, orderUsecase)

	v1.NewLoyaltyHandler(router, loyaltyUsecase)

	internal.Run(ctx, cfg, router)
	// todo use goose migrations check or up
	//if db != nil {
	//	psql.CreateOrConnectTable(db)
	//}
	//router := http3.NewRouter(repos)
	//ticker := time.NewTicker(time.Duration(cfg.Interval))
	//go repos.InteractionWithCalculationSystem(ticker, cfg.SystemAddr)
	//log.Printf("Starting server on port %s", cfg.AddressHTTP)
	//err = http.ListenAndServe(cfg.AddressHTTP, router)
	//if err != nil {
	//	log.Fatal(err)
	//}
}
func Init(ctx context.Context, addrDB string) (*db.Database, error) {
	database, err := db.NewDatabase(ctx, addrDB)
	if err != nil {
		return nil, err
	}
	return database, err
}
