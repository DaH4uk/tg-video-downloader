package users_service

import (
	"telegram-vpn-bot/internal/infrastructure/logger"
	"telegram-vpn-bot/internal/models/user_model"
	"telegram-vpn-bot/internal/repositories/users_repository"

	"github.com/pkg/errors"
)

var (
	log = logger.GetLogger()

	FailedToCreateUser   = errors.New("failed to create user")
	UserAlreadyExists    = errors.New("user already exists")
	UserAlreadyConfirmed = errors.New("user already confirmed")
	UserNotFound         = errors.New("user not found")
)

type Service struct {
	usersRepo *users_repository.Repository
}

func New(usersRepo *users_repository.Repository) *Service {
	return &Service{
		usersRepo: usersRepo,
	}
}

func (s *Service) CreateUser(user User) (*User, error) {
	//existingUser, err := s.usersRepo.GetByTelegramId(user.TelegramID)
	//
	//if err != nil {
	//	s.logger.
	//		WithField("user", user).
	//		Warn("failed to get existing user", err)
	//
	//	return nil, FailedToCreateUser
	//}
	//
	//if existingUser != nil {
	//	result := FromUserModel(existingUser)
	//
	//	return result, UserAlreadyExists
	//}

	userModel := user_model.UserModel{
		TelegramID:       user.TelegramID,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		TelegramUserName: user.TelegramUserName,
		Confirmed:        user.Confirmed,
		Admin:            user.Admin,
	}

	createResult, err := s.usersRepo.Create(userModel)
	if err != nil {
		log.
			WithField("user", user).
			WithError(err).
			Warn("failed to create user", err)

		return nil, FailedToCreateUser
	}

	result := FromUserModel(createResult)

	return result, nil
}

func (s *Service) GetAdmins() ([]user_model.UserModel, error) {
	admins, err := s.usersRepo.GetAdmins()
	if err != nil {
		log.Warn("failed to get admins", err)

		return nil, err
	}

	return admins, nil
}

func (s *Service) ConfirmUser(id uint) (*user_model.UserModel, error) {
	userModel, err := s.usersRepo.GetUserById(id)
	if err != nil {
		return nil, err
	}

	if userModel == nil {
		return nil, UserNotFound
	}

	if userModel.Confirmed {
		return userModel, UserAlreadyConfirmed
	}

	userModel.Confirmed = true

	user, err := s.usersRepo.SaveUser(userModel)
	if err != nil {
		return nil, err
	}

	return user, nil
}
