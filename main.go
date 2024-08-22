package main

import (
	"log"
	"sync"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	token = "BOT_TOEK_HERE" // Replace with your actual bot token
)

var (
	userVideos    = make(map[int64]*tgbotapi.Video)
	userQualities = make(map[int64]string)
	mu            sync.RWMutex
)

func main() {
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
			handleCallbackQuery(bot, update)
		}
	}
}
