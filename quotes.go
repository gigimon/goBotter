package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Quote struct {
	id     int
	date   string
	author string
	quote  string
}

func getQuoteFilePath(channel string) string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	curDir := filepath.Dir(ex)
	quotesFile := filepath.Join(curDir, "quotes", channel) + ".txt"

	if _, err := os.Stat(quotesFile); os.IsNotExist(err) {
		log.Printf("Quotes file %s not found\n", quotesFile)
		return ""
	}
	return quotesFile
}

func loadQuotes(channel string) []Quote {
	log.Print("Loading quotes for channel ", channel)

	quotesFile := getQuoteFilePath(channel)
	f, err := os.Open(quotesFile)

	if err != nil {
		log.Fatal(err)
		return []Quote{}
	}

	var quotes []Quote
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		res := strings.SplitN(line, " ", 6)
		id, err := strconv.Atoi(res[0])

		if err != nil {
			log.Printf("Error parsing quote id: %s\n", res[0])
			continue
		}

		q := Quote{
			id:     id,
			date:   res[1] + " " + res[2],
			author: res[4],
			quote:  res[5],
		}
		quotes = append(quotes, q)
	}
	f.Close()
	return quotes
}

func formatQuotes(quotes []Quote) string {
	var result string
	for _, q := range quotes {
		result += "[" + strconv.Itoa(q.id) + "] -> " + "[" + q.date + "] " + q.author + ": " + q.quote + "\n"
	}
	return result

}

func handleQ(ctx context.Context, b *bot.Bot, update *models.Update) {
	// loadQuotes(update.Message.Chat.Title)
	quotes := loadQuotes("#nnm")
	parts := strings.Fields(update.Message.Text)

	if len(parts) < 2 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Нужно указать номер цитаты",
		})
		return
	}

	quoteId := parts[1]
	id, err := strconv.Atoi(quoteId)

	if err != nil {
		log.Printf("Error parsing quote id: %s\n", quoteId)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Странный номер цитаты: " + quoteId,
		})
		return
	}

	for _, q := range quotes {
		if q.id == id {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   formatQuotes([]Quote{q}),
			})
			return
		}
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Цитата с номером " + quoteId + " не найдена",
	})
}

func handleAq(ctx context.Context, b *bot.Bot, update *models.Update) {
	currentTime := time.Now()

	quotes := loadQuotes("#nnm")
	quote := Quote{
		id:     quotes[len(quotes)-1].id + 1,
		date:   currentTime.Format("01/02/2006 15:04:05"),
		author: update.Message.From.Username,
		quote:  update.Message.Text[4:],
	}

	quotesFile := getQuoteFilePath("#nnm")
	f, err := os.OpenFile(quotesFile, os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatal(err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Что-то пошло не так и цитата не созранилась",
		})
		return
	}

	_, err = f.WriteString(fmt.Sprintf("%d %s %s %s %s\n", quote.id, quote.date, "#nnm", quote.author, quote.quote))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer f.Close()

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Цитата добавлена под номером: %d", quote.id),
	})
}

func handleRq(ctx context.Context, b *bot.Bot, update *models.Update) {
	quotes := loadQuotes("#nnm")

	randId := rand.Intn(len(quotes))
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   formatQuotes([]Quote{quotes[randId]}),
	})
}

func handleFq(ctx context.Context, b *bot.Bot, update *models.Update) {
	quotes := loadQuotes("#nnm")

	if len(update.Message.Text) <= 4 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Нужно указать текст для поиска",
		})
		return
	}

	searchText := update.Message.Text[4:]

	var foundQuotes []Quote

	for _, q := range quotes {
		if strings.Contains(q.quote, searchText) {
			foundQuotes = append(foundQuotes, q)
		}

		if len(foundQuotes) == 10 {
			break
		}
	}
	msg := formatQuotes(foundQuotes)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Результаты поиска [%d]:\n%s", len(foundQuotes), msg),
	})

}
