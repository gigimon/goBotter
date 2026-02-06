package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type getUpdatesResponse struct {
	OK          bool            `json:"ok"`
	Result      []models.Update `json:"result"`
	Description string          `json:"description"`
}

func startPolling(ctx context.Context, b *bot.Bot, token string) {
	var offset int64
	client := &http.Client{Timeout: 70 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		payload := map[string]any{
			"offset":  offset,
			"limit":   100,
			"timeout": 60,
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
			log.Println("Can't marshal getUpdates payload")
			log.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", token), bytes.NewReader(body))
		if err != nil {
			log.Println("Can't create getUpdates request")
			log.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Println("Can't send getUpdates request")
			log.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}

		var updates getUpdatesResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&updates)
		_ = resp.Body.Close()
		if decodeErr != nil {
			log.Println("Can't decode getUpdates response")
			log.Println(decodeErr)
			time.Sleep(1 * time.Second)
			continue
		}

		if !updates.OK {
			log.Println("Telegram getUpdates returned not ok")
			log.Println(updates.Description)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, update := range updates.Result {
			upd := update
			if upd.ID >= offset {
				offset = upd.ID + 1
			}
			b.ProcessUpdate(ctx, &upd)
		}
	}
}
