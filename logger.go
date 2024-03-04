package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func getLogsFilePath(channel string) string {
	currentTime := time.Now()
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	curDir := filepath.Dir(ex)
	logsDir := filepath.Join(curDir, "logs", channel, fmt.Sprintf("%d/%d/%d", currentTime.Year(), currentTime.Month(), currentTime.Day()))
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		err = os.MkdirAll(logsDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
	logsFile := filepath.Join(logsDir, "logs.log")

	if _, err := os.Stat(logsFile); os.IsNotExist(err) {
		f, err := os.Create(logsFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}
	return logsFile
}

func handleLogMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	var chatName string

	if update.Message.Chat.Title != "" {
		chatName = update.Message.Chat.Title
	} else if update.Message.Chat.Type == "private" {
		chatName = update.Message.Chat.Username
	}

	logPath := getLogsFilePath(chatName)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	currentTime := time.Now()
	_, err = f.WriteString(fmt.Sprintf("[%s] [%s] %s\n", currentTime.Format("15:04:05"), update.Message.From.Username, update.Message.Text))
	if err != nil {
		log.Fatal(err)
	}
}
