package users_service

import "telegram-vpn-bot/internal/models/user_model"

type User struct {
	ID               uint
	TelegramID       int64
	FirstName        string
	LastName         string
	TelegramUserName string
	Confirmed        bool
	Admin            bool
}

func (u *User) ToUserModel() user_model.UserModel {
	return user_model.UserModel{
		TelegramID:       u.TelegramID,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		TelegramUserName: u.TelegramUserName,
		Confirmed:        u.Confirmed,
		Admin:            u.Admin,
	}
}

func FromUserModel(userModel *user_model.UserModel) *User {
	return &User{
		ID:               userModel.ID,
		TelegramID:       userModel.TelegramID,
		FirstName:        userModel.FirstName,
		LastName:         userModel.LastName,
		TelegramUserName: userModel.TelegramUserName,
		Confirmed:        userModel.Confirmed,
		Admin:            userModel.Admin,
	}
}
