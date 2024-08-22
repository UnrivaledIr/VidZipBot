package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func getFileSize(filename string) (int64, error) {
	var size int64
	fi, err := os.Stat(filename)
	if err != nil {
		return size, err
	}
	size = fi.Size()
	return size, nil
}

func parseTimeToSeconds(timeStr string) float64 {
	parts := strings.Split(timeStr, ":")
	hours, _ := strconv.ParseFloat(parts[0], 64)
	minutes, _ := strconv.ParseFloat(parts[1], 64)
	seconds, _ := strconv.ParseFloat(parts[2], 64)
	return hours*3600 + minutes*60 + seconds
}

func updateProgress(bot *tgbotapi.BotAPI, chatID int64, messageID, percentage int) {
	progressBar := generateProgressBar(percentage)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, progressBar)
	_, err := bot.Send(editMsg)
	if err != nil {
		log.Println(err)
	}
}

func generateProgressBar(percentage int) string {
	progress := percentage / 10
	bar := strings.Repeat("◼️", progress) + strings.Repeat("◽️", 10-progress)
	return fmt.Sprintf("-=[%s]=- %d%%", bar, percentage)
}
