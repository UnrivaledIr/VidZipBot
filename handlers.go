package main

import (
	"bufio"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

var (
	userVideos    = make(map[int64]*tgbotapi.Video)
	userQualities = make(map[int64]string)
	userProgress  = make(map[int64]int)  // Track the progress
	mu            sync.RWMutex
)

func handleVideo(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	video := update.Message.Video

	mu.Lock()
	userVideos[update.Message.Chat.ID] = video
	mu.Unlock()

	low := "low"
	medium := "medium"
	high := "high"

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			{
				Text:         "Low Quality",
				CallbackData: &low,
			},
			{
				Text:         "Medium Quality",
				CallbackData: &medium,
			},
			{
				Text:         "High Quality",
				CallbackData: &high,
			},
		},
	}

	inlineKeyboard := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Select quality:")
	msg.ReplyMarkup = inlineKeyboard

	_, err := bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update, token string) {
	if update.CallbackQuery.Data == "show_progress" {
		mu.RLock()
		progress := userProgress[update.CallbackQuery.Message.Chat.ID]
		mu.RUnlock()

		progressMsg := fmt.Sprintf("Your job is done %d%%", progress)

		// Send a pop-up message (alert) to the user
		alert := tgbotapi.NewCallback(update.CallbackQuery.ID, progressMsg)
		alert.ShowAlert = true
		_, err := bot.AnswerCallbackQuery(alert)
		if err != nil {
			log.Println(err)
		}
		return
	}

	quality := update.CallbackQuery.Data

	mu.Lock()
	userQualities[update.CallbackQuery.Message.Chat.ID] = quality
	mu.Unlock()

	// Send a message indicating the video is compressing with a "Show progress" button
	progressButton := tgbotapi.NewInlineKeyboardButtonData("Show progress", "show_progress")
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{progressButton})

	compressingMsg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Your video is compressing ...")
	compressingMsg.ReplyMarkup = inlineKeyboard
	_, err := bot.Send(compressingMsg)
	if err != nil {
		log.Println(err)
		return
	}

	// Start video conversion in a goroutine to allow progress updates
	go convertVideo(bot, update.CallbackQuery.Message.Chat.ID, token)
}


func convertVideo(bot *tgbotapi.BotAPI, chatID int64, token string) {
	mu.RLock()
	video := userVideos[chatID]
	quality := userQualities[chatID]
	mu.RUnlock()

	fileURL, err := bot.GetFile(tgbotapi.FileConfig{FileID: video.FileID})
	if err != nil {
		log.Println(err)
		return
	}

	outputFileName := fmt.Sprintf("output_%d_%d.mp4", chatID, time.Now().UnixNano())

	var crf string
	switch quality {
	case "low":
		crf = "28"
	case "medium":
		crf = "23"
	case "high":
		crf = "18"
	default:
		crf = "23"
	}

	videoFilePath := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, fileURL.FilePath)

	cmd := exec.Command("ffmpeg", "-i", videoFilePath, "-c:v", "libx264", "-crf", crf, outputFileName, "-progress", "pipe:1", "-nostats")

	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	err = cmd.Start()
	if err != nil {
		log.Println(err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	durationRegex := regexp.MustCompile(`Duration: (\d+:\d+:\d+\.\d+)`)
	timeRegex := regexp.MustCompile(`time=(\d+:\d+:\d+\.\d+)`)

	var duration float64
	var progress int

	for scanner.Scan() {
		line := scanner.Text()

		// Extract duration from the first line that matches
		if duration == 0 {
			matches := durationRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				duration = parseTimeToSeconds(matches[1])
			}
		}

		// Extract current time and calculate progress
		matches := timeRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			currentTime := parseTimeToSeconds(matches[1])
			progress = int((currentTime / duration) * 100)
			mu.Lock()
			userProgress[chatID] = progress
			mu.Unlock()
		}
	}

	err = cmd.Wait()
	if err != nil {
		log.Println(err)
		return
	}

	mu.Lock()
	userProgress[chatID] = 100
	mu.Unlock()

	// Calculate original and compressed video sizes
	originalSize := fileURL.FileSize
	compressedSize, err := getFileSize(outputFileName)
	if err != nil {
		log.Println(err)
		return
	}

	// Calculate compression ratio
	compressionRatio := int((1 - float64(compressedSize)/float64(originalSize)) * 100)

	// Retrieve BOT_USERNAME and BOT_CHANNEL from environment variables
	botUsername := os.Getenv("BOT_USERNAME")
	botChannel := os.Getenv("BOT_CHANNEL")

	// Create caption
	caption := fmt.Sprintf("Original video size: %.2f MB\nCompressed video size: %.2f MB\nCompression ratio: %d%%\n\nBot: %s\nChannel: %s",
		float64(originalSize)/1024/1024, float64(compressedSize)/1024/1024, compressionRatio, botUsername, botChannel)

	videoMessage := tgbotapi.NewVideoUpload(chatID, outputFileName)
	videoMessage.Caption = caption
	_, err = bot.Send(videoMessage)
	if err != nil {
		log.Println(err)
	}

	mu.Lock()
	delete(userVideos, chatID)
	delete(userQualities, chatID)
	delete(userProgress, chatID)
	mu.Unlock()

	os.Remove(outputFileName)
}
