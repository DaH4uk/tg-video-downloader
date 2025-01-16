package users_service

import (
	"telegram_vpn_bot/internal/models/user_model"
	"telegram_vpn_bot/internal/repositories/users_repository"
	
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v4"
)

var (
	FailedToCreateUser = errors.New("failed to create user")
)

type Service struct {
	logger    *logrus.Logger
	usersRepo *users_repository.Repository
}

func NewUserService(logger *logrus.Logger, usersRepo *users_repository.Repository) *Service {
	return &Service{
		logger:    logger,
		usersRepo: usersRepo,
	}
}

func (s *Service) CreateUser(telegramUser telebot.User) (*user_model.UserModel, error) {
	userModel := user_model.UserModel{
		TelegramID:       telegramUser.ID,
		FirstName:        telegramUser.FirstName,
		LastName:         telegramUser.LastName,
		TelegramUserName: telegramUser.Username,
		Confirmed:        false,
		Admin:            false,
	}
	
	createResult, err := s.usersRepo.Create(userModel)
	if err != nil {
		s.logger.Warn("failed to create user", err)
		
		return nil, FailedToCreateUser
	}
	return createResult, nil
}

func (s *Service) GetAdmins() ([]user_model.UserModel, error) {
	admins, err := s.usersRepo.GetAdmins()
	if err != nil {
		s.logger.Warn("failed to get admins", err)
		
		return nil, err
	}
	
	return admins, nil
}
