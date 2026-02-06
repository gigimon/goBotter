package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var checkedChatStatus sync.Map

func ensureBotStatusLogged(ctx context.Context, b *bot.Bot, chatID int64) {
	if _, exists := checkedChatStatus.Load(chatID); exists {
		return
	}

	me, err := b.GetMe(ctx)
	if err != nil {
		log.Println("Can't get bot profile for chat status check")
		log.Println(err)
		return
	}

	member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
		ChatID: chatID,
		UserID: me.ID,
	})
	if err != nil {
		log.Println("Can't get bot chat member status")
		log.Println(err)
		return
	}

	log.Printf("Bot membership in chat %d: %s", chatID, member.Type)
	if member.Type != "administrator" && member.Type != "creator" {
		log.Printf("WARNING: bot is not admin in chat %d, reaction updates won't arrive", chatID)
	}

	checkedChatStatus.Store(chatID, true)
}

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
	log.Println("Check chat type")
	log.Println(update.Message.Chat)
	if update.Message.Chat.Type == "private" {
		chatName = update.Message.Chat.Username
	} else {
		chatName = update.Message.Chat.Title
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

func handleAllMessages(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle all messages")
	if update == nil {
		return
	}
	log.Printf(
		"Update flags: id=%d message=%t edited_message=%t callback_query=%t message_reaction=%t message_reaction_count=%t",
		update.ID,
		update.Message != nil,
		update.EditedMessage != nil,
		update.CallbackQuery != nil,
		update.MessageReaction != nil,
		update.MessageReactionCount != nil,
	)
	if dump, err := json.Marshal(update); err != nil {
		log.Println("Can't marshal update dump")
		log.Println(err)
	} else {
		log.Printf("Update dump: %s", string(dump))
	}

	if update.MessageReaction != nil {
		ensureBotStatusLogged(ctx, b, update.MessageReaction.Chat.ID)
		log.Println("Handle reaction update to stats")
		handleReactionToStats(ctx, update.MessageReaction)
		return
	}
	if update.MessageReactionCount != nil {
		ensureBotStatusLogged(ctx, b, update.MessageReactionCount.Chat.ID)
		log.Println("Handle reaction count update to stats")
		handleReactionCountToStats(ctx, update.MessageReactionCount)
		return
	}

	if update.Message == nil {
		return
	}
	if update.Message.Chat.Type != "private" {
		ensureBotStatusLogged(ctx, b, update.Message.Chat.ID)
	}

	log.Println("Log message")
	handleLogMessage(ctx, b, update)
	log.Println("Handle message to stats")
	handleMsgToStats(ctx, b, update)
}
