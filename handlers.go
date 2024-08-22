package main

import (
	"bufio"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"
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

func handleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	quality := update.CallbackQuery.Data

	mu.Lock()
	userQualities[update.CallbackQuery.Message.Chat.ID] = quality
	mu.Unlock()

	// Send a message indicating the conversion has started
	progressMsg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "-=[◽️◽️◽️◽️◽️◽️◽️◽️◽️◽️]=- 0%")
	sentMsg, err := bot.Send(progressMsg)
	if err != nil {
		log.Println(err)
		return
	}

	convertVideo(bot, update.CallbackQuery.Message.Chat.ID, sentMsg.MessageID)
}

func convertVideo(bot *tgbotapi.BotAPI, chatID int64, progressMsgID int) {
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
			progress := int((currentTime / duration) * 100)
			updateProgress(bot, chatID, progressMsgID, progress)
		}
	}

	err = cmd.Wait()
	if err != nil {
		log.Println(err)
		return
	}

	updateProgress(bot, chatID, progressMsgID, 100)

	// Calculate original and compressed video sizes
	originalSize := fileURL.FileSize
	compressedSize, err := getFileSize(outputFileName)
	if err != nil {
		log.Println(err)
		return
	}

	// Calculate compression ratio
	compressionRatio := int((1 - float64(compressedSize)/float64(originalSize)) * 100)

	// Create caption
	caption := fmt.Sprintf("Original video size: %.2f MB\nCompressed video size: %.2f MB\nCompression ratio: %d%%",
		float64(originalSize)/1024/1024, float64(compressedSize)/1024/1024, compressionRatio)

	videoMessage := tgbotapi.NewVideoUpload(chatID, outputFileName)
	videoMessage.Caption = caption
	_, err = bot.Send(videoMessage)
	if err != nil {
		log.Println(err)
	}

	mu.Lock()
	delete(userVideos, chatID)
	delete(userQualities, chatID)
	mu.Unlock()

	os.Remove(outputFileName)
}
