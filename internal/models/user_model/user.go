package user_model

import (
	"gorm.io/gorm"
)

type UserModel struct {
	gorm.Model
	TelegramID       int64 /* `gorm:"unique"` */
	FirstName        string
	LastName         string
	TelegramUserName string `gorm:"column:telegram_user_name"`
	Confirmed        bool   `gorm:"default:false;column:is_confirmed"`
	Admin            bool   `gorm:"default:false;column:is_admin"`
}
