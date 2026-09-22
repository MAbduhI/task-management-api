package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/MAbduhI/task-management-api/internal/config"
	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/pkg/password"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	logger.Info("seeding database...")

	// 1. Seed Users
	hashedPassword, err := password.Hash("password123")
	if err != nil {
		logger.Error("failed to hash password", slog.String("error", err.Error()))
		os.Exit(1)
	}

	users := []domain.User{
		{
			UUID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:         "Alice Admin",
			Email:        "alice@example.com",
			PasswordHash: hashedPassword,
		},
		{
			UUID:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Name:         "Bob Developer",
			Email:        "bob@example.com",
			PasswordHash: hashedPassword,
		},
		{
			UUID:         uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			Name:         "Charlie Backend",
			Email:        "charlie@example.com",
			PasswordHash: hashedPassword,
		},
		{
			UUID:         uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			Name:         "David Product",
			Email:        "david@example.com",
			PasswordHash: hashedPassword,
		},
	}

	for i := range users {
		var existing domain.User
		if err := db.Where("email = ?", users[i].Email).First(&existing).Error; err != nil {
			if err := db.Create(&users[i]).Error; err != nil {
				logger.Error("failed to seed user", slog.String("email", users[i].Email), slog.String("error", err.Error()))
			} else {
				logger.Info("seeded user", slog.String("email", users[i].Email), slog.String("uuid", users[i].UUID.String()))
			}
		} else {
			users[i] = existing
		}
	}

	// 2. Seed Teams
	teams := []domain.Team{
		{
			UUID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			Name: "Backend Engineering Team",
		},
		{
			UUID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			Name: "Product Management Team",
		},
	}

	for i := range teams {
		var existing domain.Team
		if err := db.Where("uuid = ?", teams[i].UUID).First(&existing).Error; err != nil {
			if err := db.Create(&teams[i]).Error; err != nil {
				logger.Error("failed to seed team", slog.String("name", teams[i].Name), slog.String("error", err.Error()))
			} else {
				logger.Info("seeded team", slog.String("name", teams[i].Name), slog.String("uuid", teams[i].UUID.String()))
			}
		} else {
			teams[i] = existing
		}
	}

	// 3. Seed Team Members
	// Backend Team: Alice (ADMIN), Bob (MEMBER), Charlie (MEMBER)
	// Product Team: David (ADMIN)
	members := []domain.TeamMember{
		{UUID: uuid.New(), TeamID: teams[0].ID, UserID: users[0].ID, Role: domain.TeamRoleAdmin},
		{UUID: uuid.New(), TeamID: teams[0].ID, UserID: users[1].ID, Role: domain.TeamRoleMember},
		{UUID: uuid.New(), TeamID: teams[0].ID, UserID: users[2].ID, Role: domain.TeamRoleMember},
		{UUID: uuid.New(), TeamID: teams[1].ID, UserID: users[3].ID, Role: domain.TeamRoleAdmin},
	}

	for _, m := range members {
		var existing domain.TeamMember
		if err := db.Where("team_id = ? AND user_id = ?", m.TeamID, m.UserID).First(&existing).Error; err != nil {
			_ = db.Create(&m)
		}
	}

	// 4. Seed Tasks
	tasks := []domain.Task{
		{
			UUID:        uuid.MustParse("9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"),
			Title:       "Setup CI/CD Pipeline with GitHub Actions",
			Description: "Automate test suite, race condition checks, and container image builds.",
			Status:      domain.TaskStatusInProgress,
			CreatedBy:   users[0].ID,
			AssigneeID:  &users[1].ID,
		},
		{
			UUID:        uuid.MustParse("8c2edb5e-4c8e-5cbe-acee-3c1e8c4eda7e"),
			Title:       "Implement Distributed Idempotency Key Lock",
			Description: "Prevent duplicate task creations with Redis atomic SETNX lock and 24h TTL.",
			Status:      domain.TaskStatusDone,
			CreatedBy:   users[0].ID,
			AssigneeID:  &users[2].ID,
		},
		{
			UUID:        uuid.MustParse("7d3fec6f-5d9f-6dcf-bdff-4d2f9d5feb8f"),
			Title:       "Database Transaction & Audit Log on Assign",
			Description: "Ensure atomic task reassignment and audit trail logging in single transaction.",
			Status:      domain.TaskStatusTodo,
			CreatedBy:   users[0].ID,
			AssigneeID:  nil,
		},
	}

	for _, t := range tasks {
		var existing domain.Task
		if err := db.Where("uuid = ?", t.UUID).First(&existing).Error; err != nil {
			if err := db.Create(&t).Error; err != nil {
				logger.Error("failed to seed task", slog.String("title", t.Title), slog.String("error", err.Error()))
			} else {
				logger.Info("seeded task", slog.String("title", t.Title), slog.String("uuid", t.UUID.String()))
			}
		}
	}

	logger.Info("database seeding completed successfully!")
	fmt.Println("\nSeed Accounts Available (Password: password123):")
	fmt.Println("1. alice@example.com   (Team: Backend Engineering, Role: ADMIN)")
	fmt.Println("2. bob@example.com     (Team: Backend Engineering, Role: MEMBER)")
	fmt.Println("3. charlie@example.com (Team: Backend Engineering, Role: MEMBER)")
	fmt.Println("4. david@example.com   (Team: Product Management, Role: ADMIN - Different Team)")
}
