package main

import (
	"log"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN is not set in .env file")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil && update.Message.Video != nil {
			handleVideo(bot, update)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update, token)
		}
	}
}
