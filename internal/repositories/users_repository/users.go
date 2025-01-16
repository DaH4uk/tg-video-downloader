package users_repository

import (
	"telegram-vpn-bot/internal/models/user_model"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewUsersRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(user user_model.UserModel) (*user_model.UserModel, error) {
	userEntity := &user
	result := r.db.Create(userEntity)

	if err := result.Error; err != nil {
		return nil, errors.Wrap(err, "error while creating user")
	}

	return userEntity, nil
}

func (r *Repository) GetByTelegramId(telegramUserId int64) (*user_model.UserModel, error) {
	user := user_model.UserModel{}
	tx := r.db.
		Where("telegram_id = ?", telegramUserId).
		First(&user)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, " error while getting user by telegram userId")
	}

	return &user, nil
}

func (r *Repository) GetAdmins() ([]user_model.UserModel, error) {
	var users []user_model.UserModel
	result := r.db.
		Where("is_admin = true").
		Find(&users)

	if err := result.Error; err != nil {
		return nil, errors.Wrap(err, "error while getting admin user list")
	}

	return users, nil
}

func (r *Repository) GetUserById(id uint) (*user_model.UserModel, error) {
	user := &user_model.UserModel{}
	tx := r.db.
		Where("id = ?", id).
		First(&user)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, " error while getting user by id")
	}

	return user, nil

}

func (r *Repository) SaveUser(user *user_model.UserModel) (*user_model.UserModel, error) {
	result := r.db.Save(user)
	if err := result.Error; err != nil {
		return nil, errors.Wrap(err, "error while updating user")
	}
	return user, nil
}
