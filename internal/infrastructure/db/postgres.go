package db

import (
	"fmt"
	"os"
	
	gormlogger "telegram-vpn-bot/internal/infrastructure/logger/gorm"
	"telegram-vpn-bot/internal/models/user_model"
	
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitPGConnection(log *logrus.Logger) (*gorm.DB, error) {
	postgresDsn := os.Getenv("POSTGRES_DSN")
	
	if postgresDsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is not set")
	}
	
	db, err := gorm.Open(postgres.New(
		postgres.Config{
			DSN:                  postgresDsn,
			PreferSimpleProtocol: true,
		}),
		&gorm.Config{
			DisableAutomaticPing: true,
			Logger:               gormlogger.New(log),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("DB connection error GORM: %w", err)
	}
	
	return db, nil
}

func MigrateDatabase(db *gorm.DB) error {
	err := db.AutoMigrate(&user_model.UserModel{})
	if err != nil {
		return errors.Wrap(err, "can't migrate UserModel")
	}
	
	return nil
}
