package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/antchfx/htmlquery"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/net/html"
)

type CurrencyValue struct {
	currency string
	value    float64
}

var currencies = map[string]map[string]string{
	"BTC": {
		"url":    "https://finance.yahoo.com/quote/BTC-USD?p=BTC-USD&.tsrc=fin-srch",
		"format": "₿ $%.2f",
	},
	"ETH": {
		"url":    "https://finance.yahoo.com/quote/ETH-USD?p=ETH-USD&.tsrc=fin-srch",
		"format": "♦ $%.2f",
	},
	"SOL": {
		"url":    "https://finance.yahoo.com/quote/SOL-USD?p=SOL-USD&.tsrc=fin-srch",
		"format": "🟣 $%.2f",
	},
	"USD": {
		"url":    "https://finance.yahoo.com/quote/RUB=X?p=RUB=X&.tsrc=fin-srch",
		"format": "💵 %.2f₽",
	},
	"EUR": {
		"url":    "https://finance.yahoo.com/quote/EURRUB=X?.tsrc=fin-srch",
		"format": "💶 %.2f₽",
	},
	"GAS": {
		"url":    "https://finance.yahoo.com/quote/TTF=F?p=TTF=F&.tsrc=fin-srch",
		"format": "⛽️ €%.2f",
	},
}

func getPage(url string) *html.Node {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Can't create request to %s (%s)", url, err)
		return nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Can't send request to %s (%s)", url, err)
		return nil
	}

	doc, err := htmlquery.Parse(resp.Body)

	if err != nil {
		log.Printf("Can't open page %s (%s)", url, err)
		return nil
	}

	return doc
}

func getValue(currency string, url string, ch chan<- CurrencyValue, wg *sync.WaitGroup) {
	defer wg.Done()

	ratio := math.Pow(10, float64(2))
	log.Printf("Get currency for %s", currency)
	body := getPage(url)
	item := htmlquery.FindOne(body, "//*[@data-testid=\"qsp-price\"]")

	value := htmlquery.SelectAttr(item, "data-value")
	log.Printf("Value for %s: %s", currency, value)

	if len(value) > 0 {
		cur, err := strconv.ParseFloat(value, 32)
		if err != nil {
			log.Printf("Can't parse to float value: %s (%s)", value, err)
		}

		if currency == "GAS" {
			cur = cur * 10
		}

		cur = math.Round(cur*ratio) / ratio
		ch <- CurrencyValue{currency, cur}
	} else {
		ch <- CurrencyValue{currency, 0}
	}
}

func handlePizdec(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Println("Handle command !пиздец")
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "_" + bot.EscapeMarkdown("Отправляю запрос к трейдерам...") + "_",
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		log.Fatal("Something went wrong on send message: ", err)
		return
	}

	order := [6]string{"BTC", "ETH", "USD", "EUR", "OIL", "GAS"}
	values := make(map[string]float64)

	var wg sync.WaitGroup
	ch := make(chan CurrencyValue, len(currencies))

	for k, v := range currencies {
		wg.Add(1)
		go getValue(k, v["url"], ch, &wg)
	}
	wg.Wait()
	close(ch)

	for res := range ch {
		log.Printf("Currency: %s, value: %.2f", res.currency, res.value)
		values[res.currency] = res.value
	}
	log.Print("All currency values processed")

	text := ""

	for c := range order {
		if values[order[c]] > 0 {
			text += fmt.Sprintf(currencies[order[c]]["format"]+"  ", values[order[c]])
		}
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})
}
