package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/MAbduhI/task-management-api/internal/config"
	delivery "github.com/MAbduhI/task-management-api/internal/delivery/http"
	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/pkg/notification"
	pgrepo "github.com/MAbduhI/task-management-api/internal/repository/postgres"
	redisrepo "github.com/MAbduhI/task-management-api/internal/repository/redis"
	"github.com/MAbduhI/task-management-api/internal/usecase"
)

func main() {
	cfg := config.Load()

	// 1. Initialize structured logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("starting task-management-api",
		slog.String("env", cfg.AppEnv),
		slog.String("port", cfg.Port),
	)

	// 2. Connect to PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		logger.Error("failed to connect to postgresql", slog.String("error", err.Error()))
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("failed to get sql.DB handle", slog.String("error", err.Error()))
		os.Exit(1)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)

	if !cfg.IsProduction() {
		if err := db.AutoMigrate(
			&domain.User{},
			&domain.Team{},
			&domain.TeamMember{},
			&domain.Task{},
			&domain.TaskLog{},
			&domain.IdempotencyRecord{},
		); err != nil {
			logger.Error("database migration failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	// 3. Connect to Redis
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn("redis connection warning (proceeding)", slog.String("error", err.Error()))
	} else {
		logger.Info("connected to redis", slog.String("host", cfg.RedisHost))
	}

	// 4. Instantiate dependencies
	txManager := pgrepo.NewGormTxManager(db)
	userRepo := pgrepo.NewUserRepo(db)
	teamRepo := pgrepo.NewTeamRepo(db)
	taskRepo := pgrepo.NewTaskRepo(db)
	taskLogRepo := pgrepo.NewTaskLogRepo(db)
	idempotencyStore := redisrepo.NewRedisIdempotencyStore(rdb)
	taskCache := redisrepo.NewRedisTaskCache(rdb)
	notifier := notification.NewLogNotifier(logger)

	authUsecase := usecase.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
	teamUsecase := usecase.NewTeamUsecase(teamRepo, userRepo)
	taskUsecase := usecase.NewTaskUsecase(
		taskRepo,
		taskLogRepo,
		userRepo,
		teamRepo,
		idempotencyStore,
		taskCache,
		txManager,
		notifier,
		cfg.IdempotencyTTL,
	)

	authHandler := delivery.NewAuthHandler(authUsecase)
	teamHandler := delivery.NewTeamHandler(teamUsecase)
	taskHandler := delivery.NewTaskHandler(taskUsecase)
	systemHandler := delivery.NewSystemHandler(db, cfg.AdminKey, cfg.IsProduction())

	router := delivery.SetupRouter(delivery.RouterConfig{
		JWTSecret:     cfg.JWTSecret,
		AuthHandler:   authHandler,
		TeamHandler:   teamHandler,
		TaskHandler:   taskHandler,
		SystemHandler: systemHandler,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server listen error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	_ = rdb.Close()
	_ = sqlDB.Close()

	logger.Info("server exited cleanly")
}
