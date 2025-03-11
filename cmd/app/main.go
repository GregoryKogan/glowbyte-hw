package main

import (
	"log/slog"
	"os"

	"github.com/GregoryKogan/glowbyte-hw/internal/config"
	"github.com/GregoryKogan/glowbyte-hw/internal/seeder"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	config.Init()

	// Connect to database
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: os.Getenv("DSN")}), &gorm.Config{
		CreateBatchSize: viper.GetInt("seeder.create_batch_size"),
		Logger:          logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		slog.Error("failed to connect database", slog.Any("error", err))
		panic(err)
	}

	// Seed database
	s := seeder.NewSeeder(db)
	s.Seed()
}
