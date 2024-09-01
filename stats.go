package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	_ "github.com/mattn/go-sqlite3"
)

type UserStat struct {
	username string
	userId   int64
	chatId   int64
	msgCount int64
	dayCount int64
	lastMsg  int
}

var statCollector = make(map[int64]map[int64]UserStat) // chat_id:user_id:UserStat

func handleMsgToStats(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle message to stats")
	if update.Message.Chat.Type == "private" {
		return
	}

	chanId := update.Message.Chat.ID
	userId := update.Message.From.ID

	if _, ok := statCollector[chanId]; !ok {
		statCollector[chanId] = make(map[int64]UserStat)
	}

	if _, ok := statCollector[chanId][userId]; !ok {
		statCollector[chanId][userId] = UserStat{
			username: update.Message.From.Username,
			userId:   userId,
			chatId:   chanId,
			msgCount: 0,
			dayCount: 0,
			lastMsg:  update.Message.Date,
		}
	}

	wordsCount := len(strings.Fields(update.Message.Text))

	userStat := statCollector[chanId][userId]
	userStat.msgCount += int64(wordsCount)

	prevDate := time.Unix(int64(userStat.lastMsg), 0)
	curDate := time.Unix(int64(update.Message.Date), 0)

	if prevDate.Day() != curDate.Day() {
		userStat.dayCount = 0
	}
	userStat.dayCount += int64(wordsCount)
	userStat.lastMsg = update.Message.Date

	statCollector[chanId][userId] = userStat
}

func saveStats() {
	log.Println("[cron] Save stats to database")
	db, err := sql.Open("sqlite3", "./stats.db")
	if err != nil {
		log.Println("Can't open stat database")
		log.Println(err)
		return
	}
	defer db.Close()

	for chatId := range statCollector {
		for userId, user := range statCollector[chatId] {
			// Get information about current user
			// Update user stat: all messages, day messages
			var dbUserStat = UserStat{}
			userExist := true

			err := db.QueryRow("SELECT chat_id, user_id, username, msg_count, day_count, last_message_date FROM message_stats WHERE chat_id = ? AND user_id = ?", chatId, userId).Scan(&dbUserStat.chatId, &dbUserStat.userId, &dbUserStat.username, &dbUserStat.msgCount, &dbUserStat.dayCount, &dbUserStat.lastMsg)
			if err != nil {
				if err != sql.ErrNoRows {
					log.Println("Can't get user stat")
					log.Println(err)
					continue
				}
				userExist = false
			}
			dbDate := time.Unix(int64(dbUserStat.lastMsg), 0)
			curDate := time.Unix(int64(user.lastMsg), 0)

			if dbDate.Day() != curDate.Day() {
				dbUserStat.dayCount = 0
			}
			dbUserStat.dayCount += user.dayCount
			dbUserStat.msgCount += user.msgCount
			dbUserStat.lastMsg = user.lastMsg
			dbUserStat.username = user.username
			dbUserStat.userId = user.userId
			dbUserStat.chatId = user.chatId

			if userExist {
				_, err = db.Exec("UPDATE message_stats SET msg_count = ?, day_count = ?, last_message_date = ? WHERE chat_id = ? AND user_id = ?", dbUserStat.msgCount, dbUserStat.dayCount, dbUserStat.lastMsg, chatId, userId)
				if err != nil {
					log.Println("Can't save user stat")
					log.Println(err)
					continue
				}
			} else {
				_, err = db.Exec("INSERT INTO message_stats (chat_id, user_id, username, msg_count, day_count, last_message_date) VALUES (?, ?, ?, ?, ?, ?)", dbUserStat.chatId, dbUserStat.userId, dbUserStat.username, dbUserStat.msgCount, dbUserStat.dayCount, dbUserStat.lastMsg)
				if err != nil {
					log.Println("Can't save user stat")
					log.Println(err)
					continue
				}
			}
			delete(statCollector[chatId], userId)
		}
	}
}

func runStatSaver() {
	if _, err := os.Stat("./stats.db"); errors.Is(err, os.ErrNotExist) {
		log.Println("Database not found, create new")
		os.Create("./stats.db")

		log.Println("Read dump file")
		dbDump, err := os.ReadFile("./stats.sql")
		if err != nil {
			log.Println("Can't read dump file")
			log.Println(err)
			return
		}

		db, err := sql.Open("sqlite3", "./stats.db")
		if err != nil {
			log.Println("Can't open stat database")
			log.Println(err)
			return
		}
		defer db.Close()

		log.Println("Create tables in database")
		_, err = db.Exec(string(dbDump))
		if err != nil {
			log.Println("Can't create tables")
			log.Println(err)
			return
		}
	}

	tick := time.Tick(10 * time.Second)
	for range tick {
		saveStats()
	}
}

func handleDayTop(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle day top")
	db, err := sql.Open("sqlite3", "./stats.db")
	if err != nil {
		log.Println("Can't open stat database")
		log.Println(err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT chat_id, user_id, username, day_count FROM message_stats WHERE chat_id = ? ORDER BY day_count DESC LIMIT 10", update.Message.Chat.ID)
	if err != nil {
		log.Println("Can't get day top")
		log.Println(err)
		return
	}
	defer rows.Close()

	msg := "Топ говорунов за день:\n"

	for rows.Next() {
		var userStat UserStat
		err := rows.Scan(&userStat.chatId, &userStat.userId, &userStat.username, &userStat.dayCount)
		if err != nil {
			log.Println("Can't scan day top")
			log.Println(err)
			continue
		}

		msg += fmt.Sprintf("%s: %d\n", userStat.username, userStat.dayCount)
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})
}

func handleTop(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle top")
	db, err := sql.Open("sqlite3", "./stats.db")
	if err != nil {
		log.Println("Can't open stat database")
		log.Println(err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT chat_id, user_id, username, msg_count FROM message_stats WHERE chat_id = ? ORDER BY msg_count DESC LIMIT 10", update.Message.Chat.ID)
	if err != nil {
		log.Println("Can't get day top")
		log.Println(err)
		return
	}
	defer rows.Close()

	msg := "Топ говорунов за всегда:\n"

	for rows.Next() {
		var userStat UserStat
		err := rows.Scan(&userStat.chatId, &userStat.userId, &userStat.username, &userStat.msgCount)
		if err != nil {
			log.Println("Can't scan day top")
			log.Println(err)
			continue
		}

		msg += fmt.Sprintf("%s: %d\n", userStat.username, userStat.msgCount)
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})
}
