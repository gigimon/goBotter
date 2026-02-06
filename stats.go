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

type ReactionStat struct {
	name  string
	count int64
}

func loadReactionStats(query string, args ...any) ([]ReactionStat, error) {
	rows, err := statsDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]ReactionStat, 0, 10)
	for rows.Next() {
		var item ReactionStat
		if err = rows.Scan(&item.name, &item.count); err != nil {
			return nil, err
		}
		stats = append(stats, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

var statsDB *sql.DB

const dayLayout = "2006-01-02"

func sendText(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	if update == nil || update.Message == nil {
		return
	}

	params := &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	}
	if update.Message.MessageThreadID != 0 {
		params.MessageThreadID = update.Message.MessageThreadID
	}

	if _, err := b.SendMessage(ctx, params); err != nil {
		log.Println("Can't send message")
		log.Println(err)
	}
}

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

		CREATE TABLE IF NOT EXISTS reaction_given_total (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			reactions_total INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS reaction_given_daily (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			day_date TEXT NOT NULL,
			username TEXT NOT NULL,
			reactions_count INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id, day_date)
		);

		CREATE TABLE IF NOT EXISTS reaction_popular_total (
			chat_id INTEGER NOT NULL,
			reaction_key TEXT NOT NULL,
			reaction_label TEXT NOT NULL,
			reactions_total INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, reaction_key)
		);

		CREATE TABLE IF NOT EXISTS reaction_popular_daily (
			chat_id INTEGER NOT NULL,
			day_date TEXT NOT NULL,
			reaction_key TEXT NOT NULL,
			reaction_label TEXT NOT NULL,
			reactions_count INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, day_date, reaction_key)
		);

		CREATE INDEX IF NOT EXISTS idx_reaction_given_daily_chat_day ON reaction_given_daily(chat_id, day_date);
		CREATE INDEX IF NOT EXISTS idx_reaction_given_total_chat ON reaction_given_total(chat_id);
		CREATE INDEX IF NOT EXISTS idx_reaction_popular_daily_chat_day ON reaction_popular_daily(chat_id, day_date);
		CREATE INDEX IF NOT EXISTS idx_reaction_popular_total_chat ON reaction_popular_total(chat_id);

		CREATE TABLE IF NOT EXISTS reaction_received_total (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			reactions_total INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS reaction_received_daily (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			day_date TEXT NOT NULL,
			username TEXT NOT NULL,
			reactions_count INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id, day_date)
		);

		CREATE INDEX IF NOT EXISTS idx_reaction_received_daily_chat_day ON reaction_received_daily(chat_id, day_date);
		CREATE INDEX IF NOT EXISTS idx_reaction_received_total_chat ON reaction_received_total(chat_id);

		CREATE TABLE IF NOT EXISTS reaction_received_type_total (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			reaction_key TEXT NOT NULL,
			reaction_label TEXT NOT NULL,
			reactions_total INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id, reaction_key)
		);

		CREATE TABLE IF NOT EXISTS reaction_received_type_daily (
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			day_date TEXT NOT NULL,
			reaction_key TEXT NOT NULL,
			reaction_label TEXT NOT NULL,
			reactions_count INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, user_id, day_date, reaction_key)
		);

		CREATE INDEX IF NOT EXISTS idx_reaction_received_type_total_chat_user ON reaction_received_type_total(chat_id, user_id);
		CREATE INDEX IF NOT EXISTS idx_reaction_received_type_daily_chat_user_day ON reaction_received_type_daily(chat_id, user_id, day_date);

		CREATE TABLE IF NOT EXISTS reaction_message_state (
			chat_id INTEGER NOT NULL,
			message_id INTEGER NOT NULL,
			reaction_key TEXT NOT NULL,
			reaction_label TEXT NOT NULL,
			last_total_count INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, message_id, reaction_key)
		);
		CREATE INDEX IF NOT EXISTS idx_reaction_message_state_chat_msg ON reaction_message_state(chat_id, message_id);

		CREATE TABLE IF NOT EXISTS message_author_state (
			chat_id INTEGER NOT NULL,
			message_id INTEGER NOT NULL,
			author_user_id INTEGER NOT NULL,
			author_name TEXT NOT NULL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(chat_id, message_id)
		);

		CREATE INDEX IF NOT EXISTS idx_message_author_state_chat_msg ON message_author_state(chat_id, message_id);
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

func reactionKeyAndLabel(reaction models.ReactionType) (string, string) {
	if reaction.ReactionTypeEmoji != nil {
		return "emoji:" + reaction.ReactionTypeEmoji.Emoji, reaction.ReactionTypeEmoji.Emoji
	}
	if reaction.ReactionTypeCustomEmoji != nil {
		customID := reaction.ReactionTypeCustomEmoji.CustomEmojiID
		return "custom:" + customID, "custom:" + customID
	}
	return "", ""
}

func reactionCounter(reactions []models.ReactionType) map[string]int {
	counter := make(map[string]int)
	for _, reaction := range reactions {
		key, _ := reactionKeyAndLabel(reaction)
		if key == "" {
			continue
		}
		counter[key]++
	}
	return counter
}

func reactorIdentity(update *models.MessageReactionUpdated) (int64, string, bool) {
	if update.User != nil {
		return update.User.ID, getUserName(update.User), true
	}
	if update.ActorChat != nil {
		return update.ActorChat.ID, update.ActorChat.Title, true
	}
	return 0, "", false
}

func getChatName(chat *models.Chat) string {
	if chat == nil {
		return ""
	}
	if chat.Title != "" {
		return chat.Title
	}
	if chat.Username != "" {
		return chat.Username
	}
	return fmt.Sprintf("chat:%d", chat.ID)
}

func getMessageAuthor(msg *models.Message) (int64, string, bool) {
	if msg == nil {
		return 0, "", false
	}
	if msg.From != nil {
		return msg.From.ID, getUserName(msg.From), true
	}
	if msg.SenderChat != nil {
		return msg.SenderChat.ID, getChatName(msg.SenderChat), true
	}
	return 0, "", false
}

func upsertMessageAuthorState(tx *sql.Tx, chatID int64, messageID int, authorID int64, authorName string, updatedAt int) error {
	_, err := tx.Exec(`
		INSERT INTO message_author_state(chat_id, message_id, author_user_id, author_name, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, message_id) DO UPDATE SET
			author_user_id = excluded.author_user_id,
			author_name = excluded.author_name,
			updated_at = excluded.updated_at
	`, chatID, messageID, authorID, authorName, updatedAt)
	return err
}

func getMessageAuthorState(tx *sql.Tx, chatID int64, messageID int) (int64, string, bool, error) {
	var receiverID int64
	var receiverName string
	err := tx.QueryRow("SELECT author_user_id, author_name FROM message_author_state WHERE chat_id = ? AND message_id = ?", chatID, messageID).Scan(&receiverID, &receiverName)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", false, nil
		}
		return 0, "", false, err
	}
	return receiverID, receiverName, true, nil
}

func upsertReactionReceived(tx *sql.Tx, chatID int64, receiverID int64, receiverName string, dayDate string, totalDelta int, updatedAt int) error {
	if totalDelta <= 0 {
		return nil
	}
	if _, err := tx.Exec(`
		INSERT INTO reaction_received_total(chat_id, user_id, username, reactions_total, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id) DO UPDATE SET
			username = excluded.username,
			reactions_total = reaction_received_total.reactions_total + excluded.reactions_total,
			updated_at = excluded.updated_at
	`, chatID, receiverID, receiverName, totalDelta, updatedAt); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO reaction_received_daily(chat_id, user_id, day_date, username, reactions_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id, day_date) DO UPDATE SET
			username = excluded.username,
			reactions_count = reaction_received_daily.reactions_count + excluded.reactions_count,
			updated_at = excluded.updated_at
	`, chatID, receiverID, dayDate, receiverName, totalDelta, updatedAt); err != nil {
		return err
	}
	return nil
}

func upsertReactionReceivedByType(tx *sql.Tx, chatID int64, receiverID int64, dayDate string, reactionKey string, reactionLabel string, delta int, updatedAt int) error {
	if delta <= 0 {
		return nil
	}
	if _, err := tx.Exec(`
		INSERT INTO reaction_received_type_total(chat_id, user_id, reaction_key, reaction_label, reactions_total, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id, reaction_key) DO UPDATE SET
			reaction_label = excluded.reaction_label,
			reactions_total = reaction_received_type_total.reactions_total + excluded.reactions_total,
			updated_at = excluded.updated_at
	`, chatID, receiverID, reactionKey, reactionLabel, delta, updatedAt); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO reaction_received_type_daily(chat_id, user_id, day_date, reaction_key, reaction_label, reactions_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id, day_date, reaction_key) DO UPDATE SET
			reaction_label = excluded.reaction_label,
			reactions_count = reaction_received_type_daily.reactions_count + excluded.reactions_count,
			updated_at = excluded.updated_at
	`, chatID, receiverID, dayDate, reactionKey, reactionLabel, delta, updatedAt); err != nil {
		return err
	}
	return nil
}

func handleReactionToStats(ctx context.Context, update *models.MessageReactionUpdated) {
	log.Println("Handle reaction to stats")
	if update == nil {
		return
	}
	if update.Chat.Type == "private" {
		return
	}
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	userID, username, ok := reactorIdentity(update)
	if !ok {
		return
	}

	oldCounter := reactionCounter(update.OldReaction)
	newCounter := reactionCounter(update.NewReaction)

	addedTotal := 0
	addedByKey := make(map[string]int)
	addedLabelByKey := make(map[string]string)

	for _, reaction := range update.NewReaction {
		key, label := reactionKeyAndLabel(reaction)
		if key == "" {
			continue
		}
		addedLabelByKey[key] = label
	}

	for key, newCount := range newCounter {
		diff := newCount - oldCounter[key]
		if diff <= 0 {
			continue
		}
		addedByKey[key] = diff
		addedTotal += diff
	}

	if addedTotal == 0 {
		return
	}

	chatID := update.Chat.ID
	msgDate := update.Date
	dayDate := time.Unix(int64(msgDate), 0).In(time.Local).Format(dayLayout)

	tx, err := statsDB.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Can't start reaction stats transaction")
		log.Println(err)
		return
	}

	receiverID, receiverName, hasReceiver, err := getMessageAuthorState(tx, chatID, update.MessageID)
	if err != nil {
		_ = tx.Rollback()
		log.Println("Can't read message author state for reaction update")
		log.Println(err)
		return
	}

	if _, err = tx.Exec(`
		INSERT INTO reaction_given_total(chat_id, user_id, username, reactions_total, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id) DO UPDATE SET
			username = excluded.username,
			reactions_total = reaction_given_total.reactions_total + excluded.reactions_total,
			updated_at = excluded.updated_at
	`, chatID, userID, username, addedTotal, msgDate); err != nil {
		_ = tx.Rollback()
		log.Println("Can't save total given reactions stats")
		log.Println(err)
		return
	}

	if _, err = tx.Exec(`
		INSERT INTO reaction_given_daily(chat_id, user_id, day_date, username, reactions_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, user_id, day_date) DO UPDATE SET
			username = excluded.username,
			reactions_count = reaction_given_daily.reactions_count + excluded.reactions_count,
			updated_at = excluded.updated_at
	`, chatID, userID, dayDate, username, addedTotal, msgDate); err != nil {
		_ = tx.Rollback()
		log.Println("Can't save daily given reactions stats")
		log.Println(err)
		return
	}

	for key, delta := range addedByKey {
		label := addedLabelByKey[key]
		if _, err = tx.Exec(`
			INSERT INTO reaction_popular_total(chat_id, reaction_key, reaction_label, reactions_total, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(chat_id, reaction_key) DO UPDATE SET
				reaction_label = excluded.reaction_label,
				reactions_total = reaction_popular_total.reactions_total + excluded.reactions_total,
				updated_at = excluded.updated_at
		`, chatID, key, label, delta, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't save total popular reaction stats")
			log.Println(err)
			return
		}

		if _, err = tx.Exec(`
			INSERT INTO reaction_popular_daily(chat_id, day_date, reaction_key, reaction_label, reactions_count, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(chat_id, day_date, reaction_key) DO UPDATE SET
				reaction_label = excluded.reaction_label,
				reactions_count = reaction_popular_daily.reactions_count + excluded.reactions_count,
				updated_at = excluded.updated_at
		`, chatID, dayDate, key, label, delta, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't save daily popular reaction stats")
			log.Println(err)
			return
		}

		if hasReceiver {
			if err = upsertReactionReceivedByType(tx, chatID, receiverID, dayDate, key, label, delta, msgDate); err != nil {
				_ = tx.Rollback()
				log.Println("Can't save received reactions by type stats")
				log.Println(err)
				return
			}
		}
	}

	if hasReceiver {
		if err = upsertReactionReceived(tx, chatID, receiverID, receiverName, dayDate, addedTotal, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't save received reactions stats")
			log.Println(err)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		log.Println("Can't commit reaction stats transaction")
		log.Println(err)
	}
}

func handleReactionCountToStats(ctx context.Context, update *models.MessageReactionCountUpdated) {
	log.Println("Handle reaction count to stats")
	if update == nil {
		return
	}
	if update.Chat.Type == "private" {
		return
	}
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatID := update.Chat.ID
	messageID := update.MessageID
	msgDate := update.Date
	dayDate := time.Unix(int64(msgDate), 0).In(time.Local).Format(dayLayout)

	tx, err := statsDB.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Can't start reaction count stats transaction")
		log.Println(err)
		return
	}

	receiverID, receiverName, hasReceiver, err := getMessageAuthorState(tx, chatID, messageID)
	if err != nil {
		_ = tx.Rollback()
		log.Println("Can't read message author state for reaction count update")
		log.Println(err)
		return
	}

	addedTotal := 0

	for _, reactionCount := range update.Reactions {
		key, label := reactionKeyAndLabel(reactionCount.Type)
		if key == "" {
			continue
		}

		var prevCount int64
		queryErr := tx.QueryRow("SELECT last_total_count FROM reaction_message_state WHERE chat_id = ? AND message_id = ? AND reaction_key = ?", chatID, messageID, key).Scan(&prevCount)
		if queryErr != nil && queryErr != sql.ErrNoRows {
			_ = tx.Rollback()
			log.Println("Can't read reaction message state")
			log.Println(queryErr)
			return
		}

		newCount := int64(reactionCount.TotalCount)
		delta := newCount - prevCount

		if _, err = tx.Exec(`
			INSERT INTO reaction_message_state(chat_id, message_id, reaction_key, reaction_label, last_total_count, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(chat_id, message_id, reaction_key) DO UPDATE SET
				reaction_label = excluded.reaction_label,
				last_total_count = excluded.last_total_count,
				updated_at = excluded.updated_at
		`, chatID, messageID, key, label, newCount, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't upsert reaction message state")
			log.Println(err)
			return
		}

		if delta <= 0 {
			continue
		}
		addedTotal += int(delta)

		if _, err = tx.Exec(`
			INSERT INTO reaction_popular_total(chat_id, reaction_key, reaction_label, reactions_total, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(chat_id, reaction_key) DO UPDATE SET
				reaction_label = excluded.reaction_label,
				reactions_total = reaction_popular_total.reactions_total + excluded.reactions_total,
				updated_at = excluded.updated_at
		`, chatID, key, label, delta, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't save total popular reaction stats from count update")
			log.Println(err)
			return
		}

		if _, err = tx.Exec(`
			INSERT INTO reaction_popular_daily(chat_id, day_date, reaction_key, reaction_label, reactions_count, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(chat_id, day_date, reaction_key) DO UPDATE SET
				reaction_label = excluded.reaction_label,
				reactions_count = reaction_popular_daily.reactions_count + excluded.reactions_count,
				updated_at = excluded.updated_at
		`, chatID, dayDate, key, label, delta, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't save daily popular reaction stats from count update")
			log.Println(err)
			return
		}

		if hasReceiver {
			if err = upsertReactionReceivedByType(tx, chatID, receiverID, dayDate, key, label, int(delta), msgDate); err != nil {
				_ = tx.Rollback()
				log.Println("Can't save received reactions by type stats from count update")
				log.Println(err)
				return
			}
		}
	}

	if hasReceiver {
		if err = upsertReactionReceived(tx, chatID, receiverID, receiverName, dayDate, addedTotal, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't save received reactions stats from count update")
			log.Println(err)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		log.Println("Can't commit reaction count stats transaction")
		log.Println(err)
	}
}

func handleReactionUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle reaction update via match-func")
	handleReactionToStats(ctx, update.MessageReaction)
}

func handleReactionCountUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle reaction count update via match-func")
	handleReactionCountToStats(ctx, update.MessageReactionCount)
}

func handleMsgToStats(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle message to stats")
	if update.Message.Chat.Type == "private" {
		return
	}

	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatID := update.Message.Chat.ID
	authorID, authorName, hasAuthor := getMessageAuthor(update.Message)
	if !hasAuthor {
		return
	}

	msgDate := update.Message.Date
	dayDate := time.Unix(int64(msgDate), 0).In(time.Local).Format(dayLayout)
	wordsCount := len(strings.Fields(update.Message.Text))

	tx, err := statsDB.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Can't start stats transaction")
		log.Println(err)
		return
	}

	if err = upsertMessageAuthorState(tx, chatID, update.Message.ID, authorID, authorName, msgDate); err != nil {
		_ = tx.Rollback()
		log.Println("Can't save message author state")
		log.Println(err)
		return
	}

	if wordsCount > 0 {
		if _, err = tx.Exec(`
			INSERT INTO stats_total(chat_id, user_id, username, words_total, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(chat_id, user_id) DO UPDATE SET
				username = excluded.username,
				words_total = stats_total.words_total + excluded.words_total,
				updated_at = excluded.updated_at
		`, chatID, authorID, authorName, wordsCount, msgDate); err != nil {
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
		`, chatID, authorID, dayDate, authorName, wordsCount, msgDate); err != nil {
			_ = tx.Rollback()
			log.Println("Can't save daily stats")
			log.Println(err)
			return
		}
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

	sendText(ctx, b, update, msg)
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

	sendText(ctx, b, update, msg)
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
	sendText(ctx, b, update, msg)
}

func handleReactionDayTop(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle reaction day top")
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatID := update.Message.Chat.ID
	today := time.Now().In(time.Local).Format(dayLayout)

	userStats, err := loadReactionStats("SELECT username, reactions_count FROM reaction_given_daily WHERE chat_id = ? AND day_date = ? ORDER BY reactions_count DESC LIMIT 10", chatID, today)
	if err != nil {
		log.Println("Can't get day top by users reactions")
		log.Println(err)
		return
	}

	reactionStats, err := loadReactionStats("SELECT reaction_label, reactions_count FROM reaction_popular_daily WHERE chat_id = ? AND day_date = ? ORDER BY reactions_count DESC LIMIT 10", chatID, today)
	if err != nil {
		log.Println("Can't get day top popular reactions")
		log.Println(err)
		return
	}
	receivedStats, err := loadReactionStats("SELECT username, reactions_count FROM reaction_received_daily WHERE chat_id = ? AND day_date = ? ORDER BY reactions_count DESC LIMIT 10", chatID, today)
	if err != nil {
		log.Println("Can't get day top by received reactions")
		log.Println(err)
		return
	}

	msg := "Реакции за день:\n\nТоп кто ставил:\n"
	place := 1
	for _, item := range userStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных (нужны новые сообщения после обновления статистики)\n"
	}

	msg += "\nТоп кому ставили:\n"
	place = 1
	for _, item := range receivedStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных (нужны новые сообщения после обновления статистики)\n"
	}

	msg += "\nТоп популярных реакций:\n"
	place = 1
	for _, item := range reactionStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных"
	}

	sendText(ctx, b, update, msg)
}

func handleReactionTop(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle reaction top")
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatID := update.Message.Chat.ID

	userStats, err := loadReactionStats("SELECT username, reactions_total FROM reaction_given_total WHERE chat_id = ? ORDER BY reactions_total DESC LIMIT 10", chatID)
	if err != nil {
		log.Println("Can't get all-time top by users reactions")
		log.Println(err)
		return
	}

	reactionStats, err := loadReactionStats("SELECT reaction_label, reactions_total FROM reaction_popular_total WHERE chat_id = ? ORDER BY reactions_total DESC LIMIT 10", chatID)
	if err != nil {
		log.Println("Can't get all-time top popular reactions")
		log.Println(err)
		return
	}
	receivedStats, err := loadReactionStats("SELECT username, reactions_total FROM reaction_received_total WHERE chat_id = ? ORDER BY reactions_total DESC LIMIT 10", chatID)
	if err != nil {
		log.Println("Can't get all-time top by received reactions")
		log.Println(err)
		return
	}

	msg := "Реакции за всё время:\n\nТоп кто ставил:\n"
	place := 1
	for _, item := range userStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных\n"
	}

	msg += "\nТоп кому ставили:\n"
	place = 1
	for _, item := range receivedStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных\n"
	}

	msg += "\nТоп популярных реакций:\n"
	place = 1
	for _, item := range reactionStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных"
	}

	sendText(ctx, b, update, msg)
}

func handleMyReaction(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle my reaction")
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	today := time.Now().In(time.Local).Format(dayLayout)

	var todayCount int64
	err := statsDB.QueryRow("SELECT COALESCE(reactions_count, 0) FROM reaction_given_daily WHERE chat_id = ? AND user_id = ? AND day_date = ?", chatID, userID, today).Scan(&todayCount)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Can't get daily reaction stat")
		log.Println(err)
		return
	}

	var totalCount int64
	err = statsDB.QueryRow("SELECT COALESCE(reactions_total, 0) FROM reaction_given_total WHERE chat_id = ? AND user_id = ?", chatID, userID).Scan(&totalCount)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Can't get total reaction stat")
		log.Println(err)
		return
	}

	name := getUserName(update.Message.From)
	msg := fmt.Sprintf("%s, твои реакции:\nСегодня добавлено: %d\nЗа всё время добавлено: %d", name, todayCount, totalCount)
	sendText(ctx, b, update, msg)
}

func handleMyReceivedReaction(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle my received reaction")
	if statsDB == nil {
		log.Println("stats database is not initialized")
		return
	}

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	today := time.Now().In(time.Local).Format(dayLayout)

	var todayCount int64
	err := statsDB.QueryRow("SELECT COALESCE(reactions_count, 0) FROM reaction_received_daily WHERE chat_id = ? AND user_id = ? AND day_date = ?", chatID, userID, today).Scan(&todayCount)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Can't get daily received reaction stat")
		log.Println(err)
		return
	}

	var totalCount int64
	err = statsDB.QueryRow("SELECT COALESCE(reactions_total, 0) FROM reaction_received_total WHERE chat_id = ? AND user_id = ?", chatID, userID).Scan(&totalCount)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Can't get total received reaction stat")
		log.Println(err)
		return
	}

	name := getUserName(update.Message.From)
	dayTypeStats, err := loadReactionStats("SELECT reaction_label, reactions_count FROM reaction_received_type_daily WHERE chat_id = ? AND user_id = ? AND day_date = ? ORDER BY reactions_count DESC LIMIT 5", chatID, userID, today)
	if err != nil {
		log.Println("Can't get daily received reaction by type stat")
		log.Println(err)
		return
	}

	totalTypeStats, err := loadReactionStats("SELECT reaction_label, reactions_total FROM reaction_received_type_total WHERE chat_id = ? AND user_id = ? ORDER BY reactions_total DESC LIMIT 5", chatID, userID)
	if err != nil {
		log.Println("Can't get total received reaction by type stat")
		log.Println(err)
		return
	}

	msg := fmt.Sprintf("%s, твои полученные реакции:\nСегодня получено: %d\nЗа всё время получено: %d", name, todayCount, totalCount)
	msg += "\n\nСегодня по типам:\n"
	place := 1
	for _, item := range dayTypeStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных\n"
	}

	msg += "\nЗа всё время по типам:\n"
	place = 1
	for _, item := range totalTypeStats {
		msg += fmt.Sprintf("%d. %s: %d\n", place, item.name, item.count)
		place++
	}
	if place == 1 {
		msg += "Пока нет данных"
	}

	sendText(ctx, b, update, msg)
}
