package infrastructure

import (
	"telegram-vpn-bot/internal/infrastructure/db"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Infra struct {
	DB *gorm.DB
}

func Init() (*Infra, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load .env file")
	}

	dbConnection, err := db.InitDB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init database")
	}

	err = db.MigrateDatabase(dbConnection)
	if err != nil {
		return nil, errors.Wrap(err, "failed to migrate database")
	}

	return &Infra{
		DB: dbConnection,
	}, nil
}
