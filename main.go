package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-telegram/bot"
)

type telegramAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func configureAllowedUpdates(token string) error {
	payload := map[string]any{
		"offset":  -1,
		"limit":   1,
		"timeout": 0,
		"allowed_updates": []string{
			"message",
			"edited_message",
			"callback_query",
			"message_reaction",
			"message_reaction_count",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("can't marshal allowed updates payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", token), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("can't create getUpdates request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("can't send getUpdates request: %w", err)
	}
	defer resp.Body.Close()

	var result telegramAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("can't decode getUpdates response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram getUpdates returned not ok: %s", result.Description)
	}

	return nil
}

func main() {
	log.Println("Start application")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handleAllMessages),
	}

	token := os.Getenv("TELEGRAM_API_TOKEN")

	if err := configureAllowedUpdates(token); err != nil {
		log.Println("Warning: can't configure allowed updates for reactions")
		log.Println(err)
	}

	debug := os.Getenv("DEBUG")
	if len(debug) != 0 {
		opts = append(opts, bot.WithDebug())
	}

	goBotter, err := bot.New(token, opts...)
	if err != nil {
		log.Println("Can't create new bot instance")
		log.Println(err)
		os.Exit(1)
	}

	err = initStatsStorage()
	if err != nil {
		log.Println("Can't initialize stats storage")
		log.Println(err)
		os.Exit(1)
	}

	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!пиздец", bot.MatchTypeExact, handlePizdec)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!q", bot.MatchTypePrefix, handleQ)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!rq", bot.MatchTypeExact, handleRq)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!aq", bot.MatchTypePrefix, handleAq)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!fq", bot.MatchTypePrefix, handleFq)

	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топдень", bot.MatchTypeExact, handleDayTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топ", bot.MatchTypeExact, handleTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!моястата", bot.MatchTypeExact, handleMyStat)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топреакдень", bot.MatchTypeExact, handleReactionDayTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топреак", bot.MatchTypeExact, handleReactionTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топреакт", bot.MatchTypeExact, handleReactionTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!мояреак", bot.MatchTypeExact, handleMyReaction)

	log.Println("Start bot")
	goBotter.Start(ctx)
}
