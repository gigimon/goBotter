package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	_ "github.com/mattn/go-sqlite3"
)

type UserStat struct {
	username string
	msgCount int64
	dayCount int64
}

var statsDB *sql.DB

const dayLayout = "2006-01-02"

func initStatsStorage() error {
	db, err := sql.Open("sqlite3", "file:stats.db?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("can't open stat database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS stats_total (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			words_total INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS stats_daily (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			day_date TEXT NOT NULL,
			username TEXT NOT NULL,
			words_count INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id, day_date)
		);

		CREATE INDEX IF NOT EXISTS idx_stats_daily_chat_day ON stats_daily(chat_id, day_date);
		CREATE INDEX IF NOT EXISTS idx_stats_total_chat ON stats_total(chat_id);
	`); err != nil {
		db.Close()
		return fmt.Errorf("can't create stat tables: %w", err)
	}

	if err = migrateLegacyStats(db); err != nil {
		db.Close()
		return err
	}

	statsDB = db
	return nil
}

func migrateLegacyStats(db *sql.DB) error {
	var applied int
	err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations WHERE name = 'v2_stats_tables'").Scan(&applied)
	if err != nil {
		return fmt.Errorf("can't check migrations: %w", err)
	}

	if applied > 0 {
		return nil
	}

	var legacyExists int
	err = db.QueryRow("SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'message_stats'").Scan(&legacyExists)
	if err != nil {
		return fmt.Errorf("can't check legacy stats table: %w", err)
	}

	if legacyExists > 0 {
		if _, err = db.Exec(`
			INSERT INTO stats_total(chat_id, user_id, username, words_total, updated_at)
			SELECT
				chat_id,
				user_id,
				COALESCE(MAX(NULLIF(username, '')), CAST(user_id AS TEXT)) AS username,
				SUM(COALESCE(msg_count, 0)) AS words_total,
				MAX(COALESCE(CAST(last_message_date AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER))) AS updated_at
			FROM message_stats
			GROUP BY chat_id, user_id
			ON CONFLICT(chat_id, user_id) DO UPDATE SET
				username = excluded.username,
				words_total = excluded.words_total,
				updated_at = excluded.updated_at
		`); err != nil {
			return fmt.Errorf("can't migrate total stats: %w", err)
		}

		if _, err = db.Exec(`
			INSERT INTO stats_daily(chat_id, user_id, day_date, username, words_count, updated_at)
			SELECT
				chat_id,
				user_id,
				DATE(COALESCE(CAST(last_message_date AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER)), 'unixepoch', 'localtime') AS day_date,
				COALESCE(MAX(NULLIF(username, '')), CAST(user_id AS TEXT)) AS username,
				SUM(COALESCE(day_count, 0)) AS words_count,
				MAX(COALESCE(CAST(last_message_date AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER))) AS updated_at
			FROM message_stats
			GROUP BY chat_id, user_id, day_date
			ON CONFLICT(chat_id, user_id, day_date) DO UPDATE SET
				username = excluded.username,
				words_count = excluded.words_count,
				updated_at = excluded.updated_at
		`); err != nil {
			return fmt.Errorf("can't migrate daily stats: %w", err)
		}
	}

	if _, err = db.Exec("INSERT INTO schema_migrations(name, applied_at) VALUES('v2_stats_tables', ?)", time.Now().Unix()); err != nil {
		return fmt.Errorf("can't save migration mark: %w", err)
	}

	return nil
}

func getUserName(from *models.User) string {
	if from.Username != "" {
		return from.Username
	}
	fullName := strings.TrimSpace(strings.TrimSpace(from.FirstName + " " + from.LastName))
	if fullName != "" {
		return fullName
	}
	return fmt.Sprintf("id:%d", from.ID)
}

func handleMsgToStats(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle message to stats")
	if update.Message.Chat.Type == "private" {
		return
	}

	wordsCount := len(strings.Fields(update.Message.Text))
	if wordsCount == 0 {
		return
	}

	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatId := update.Message.Chat.ID
	userId := update.Message.From.ID
	username := getUserName(update.Message.From)
	msgDate := update.Message.Date
	dayDate := time.Unix(int64(msgDate), 0).In(time.Local).Format(dayLayout)

	tx, err := statsDB.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Can't start stats transaction")
		log.Println(err)
		return
	}

	if _, err = tx.Exec(`
		INSERT INTO stats_total(chat_id, user_id, username, words_total, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id) DO UPDATE SET
			username = excluded.username,
			words_total = stats_total.words_total + excluded.words_total,
			updated_at = excluded.updated_at
	`, chatId, userId, username, wordsCount, msgDate); err != nil {
		_ = tx.Rollback()
		log.Println("Can't save total stats")
		log.Println(err)
		return
	}

	if _, err = tx.Exec(`
		INSERT INTO stats_daily(chat_id, user_id, day_date, username, words_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id, day_date) DO UPDATE SET
			username = excluded.username,
			words_count = stats_daily.words_count + excluded.words_count,
			updated_at = excluded.updated_at
	`, chatId, userId, dayDate, username, wordsCount, msgDate); err != nil {
		_ = tx.Rollback()
		log.Println("Can't save daily stats")
		log.Println(err)
		return
	}

	if err = tx.Commit(); err != nil {
		log.Println("Can't commit stats transaction")
		log.Println(err)
	}
}

func handleDayTop(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle day top")
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	today := time.Now().In(time.Local).Format(dayLayout)

	rows, err := statsDB.Query("SELECT username, words_count FROM stats_daily WHERE chat_id = ? AND day_date = ? ORDER BY words_count DESC LIMIT 10", update.Message.Chat.ID, today)
	if err != nil {
		log.Println("Can't get day top")
		log.Println(err)
		return
	}
	defer rows.Close()

	msg := "Топ говорунов за день (слов):\n"
	place := 1

	for rows.Next() {
		var userStat UserStat
		err := rows.Scan(&userStat.username, &userStat.dayCount)
		if err != nil {
			log.Println("Can't scan day top")
			log.Println(err)
			continue
		}

		msg += fmt.Sprintf("%d. %s: %d\n", place, userStat.username, userStat.dayCount)
		place++
	}

	if place == 1 {
		msg += "Сегодня сообщений пока нет"
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})
}

func handleTop(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle top")
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	rows, err := statsDB.Query("SELECT username, words_total FROM stats_total WHERE chat_id = ? ORDER BY words_total DESC LIMIT 10", update.Message.Chat.ID)
	if err != nil {
		log.Println("Can't get day top")
		log.Println(err)
		return
	}
	defer rows.Close()

	msg := "Топ говорунов за всё время (слов):\n"
	place := 1

	for rows.Next() {
		var userStat UserStat
		err := rows.Scan(&userStat.username, &userStat.msgCount)
		if err != nil {
			log.Println("Can't scan day top")
			log.Println(err)
			continue
		}

		msg += fmt.Sprintf("%d. %s: %d\n", place, userStat.username, userStat.msgCount)
		place++
	}

	if place == 1 {
		msg += "Сообщений пока нет"
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})
}

func handleMyStat(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle my stat")
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatId := update.Message.Chat.ID
	userId := update.Message.From.ID
	today := time.Now().In(time.Local).Format(dayLayout)

	var todayWords int64
	err := statsDB.QueryRow("SELECT COALESCE(words_count, 0) FROM stats_daily WHERE chat_id = ? AND user_id = ? AND day_date = ?", chatId, userId, today).Scan(&todayWords)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Can't get daily user stat")
		log.Println(err)
		return
	}

	var totalWords int64
	err = statsDB.QueryRow("SELECT COALESCE(words_total, 0) FROM stats_total WHERE chat_id = ? AND user_id = ?", chatId, userId).Scan(&totalWords)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Can't get total user stat")
		log.Println(err)
		return
	}

	name := getUserName(update.Message.From)
	msg := fmt.Sprintf("%s, твоя статистика (слов):\nСегодня: %d\nЗа всё время: %d", name, todayWords, totalWords)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatId,
		Text:   msg,
	})
}
