package main

import (
	"github.com/pkg/errors"

	"telegram-vpn-bot/internal/handlers"
	cancelusercallback "telegram-vpn-bot/internal/handlers/callback/ban_user"
	confirmusercallback "telegram-vpn-bot/internal/handlers/callback/confirm_user"
	startcommand "telegram-vpn-bot/internal/handlers/command/start"
	"telegram-vpn-bot/internal/infrastructure"
	"telegram-vpn-bot/internal/infrastructure/logger"
	"telegram-vpn-bot/internal/repositories/users_repository"
	"telegram-vpn-bot/internal/services/message_handler"
	"telegram-vpn-bot/internal/services/notifier_service"
	"telegram-vpn-bot/internal/services/users_service"
)

func main() {
	log := logger.GetLogger()

	infra, err := infrastructure.Init()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to init infrastructure"))
	}

	usersRepository := users_repository.NewUsersRepository(infra.DB)

	telegramBotApi, _ := handlers.InitBotApi()

	usersService := users_service.New(usersRepository)
	notifierService := notifier_service.New(telegramBotApi, usersService)

	handler := message_handler.New(telegramBotApi)

	handler.RegisterCommandHandler("start", startcommand.NewStartHandler(telegramBotApi, notifierService, usersService))

	handler.RegisterCallback("confirm_user", confirmusercallback.New(telegramBotApi, usersService))
	handler.RegisterCallback("cancel_user", cancelusercallback.New(telegramBotApi, usersService))

	handler.HandleUpdates()
}
