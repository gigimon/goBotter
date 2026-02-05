package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type CurrencyValue struct {
	currency string
	value    float64
}

var currencies = map[string]map[string]string{
	"BTC": {
		"type":   "crypto",
		"format": "₿ $%.2f",
		"key":    "bitcoin",
	},
	"ETH": {
		"type":   "crypto",
		"format": "♦ $%.2f",
		"key":    "ethereum",
	},
	"SOL": {
		"type":   "crypto",
		"format": "☀️ $%.2f",
		"key":    "solana",
	},
	"USD": {
		"type":   "currency",
		"format": "💵 %.2f₽",
	},
	"EUR": {
		"type":   "currency",
		"format": "💶 %.2f₽",
	},
	"CNY": {
		"type":   "currency",
		"format": "💴 %.2f₽",
	},
}

func getCryptoValues(ch chan CurrencyValue, wg *sync.WaitGroup) {
	defer wg.Done()
	req, err := http.NewRequest(http.MethodGet, "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin%2Csolana%2Cethereum&vs_currencies=usd&include_market_cap=false&include_24hr_vol=false&include_24hr_change=false&include_last_updated_at=false", nil)
	if err != nil {
		log.Printf("Can't get currencies from coingecko", err)
		return
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Can't send request to %s", err)
		return
	}
	defer resp.Body.Close()
	var prices map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		log.Fatalf("Fail to parse JSON: %v", err)
	}
	for k, v := range currencies {
		if v["type"] == "crypto" {
			ch <- CurrencyValue{currency: k, value: prices[v["key"]]["usd"]}
		}
	}
}

func getCurrencyPrices(ch chan CurrencyValue, wg *sync.WaitGroup) {
	defer wg.Done()
	
	// Запрос к новому API
	req, err := http.NewRequest(http.MethodGet, "https://informers.forexpf.ru/export/euusrub.js", nil)
	if err != nil {
		log.Printf("Ошибка создания запроса: %v", err)
		return
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Ошибка отправки запроса: %v", err)
		return
	}
	defer resp.Body.Close()
	
	// Чтение ответа как текст
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		log.Printf("Ошибка чтения ответа: %v", err)
		return
	}
	
	responseText := buf.String()
	
	// Извлекаем значения курсов с помощью регулярных выражений
	currencyValues := map[string]float64{
		"USD": 0,
		"EUR": 0,
		"CNY": 0,
	}
	
	// Регулярное выражение для извлечения значений
	usdRegex := regexp.MustCompile(`document\.getElementById\("usdrubbid"\)\.innerHTML=([0-9\.]+);`)
	eurRegex := regexp.MustCompile(`document\.getElementById\("eurrubbid"\)\.innerHTML=([0-9\.]+);`)
	cnyRegex := regexp.MustCompile(`document\.getElementById\("cnyrubbid"\)\.innerHTML=([0-9\.]+);`)
	
	// Извлечение значений
	if usdMatches := usdRegex.FindStringSubmatch(responseText); len(usdMatches) > 1 {
		usdValue, err := strconv.ParseFloat(usdMatches[1], 64)
		if err == nil {
			currencyValues["USD"] = usdValue
		}
	}
	
	if eurMatches := eurRegex.FindStringSubmatch(responseText); len(eurMatches) > 1 {
		eurValue, err := strconv.ParseFloat(eurMatches[1], 64)
		if err == nil {
			currencyValues["EUR"] = eurValue
		}
	}
	
	if cnyMatches := cnyRegex.FindStringSubmatch(responseText); len(cnyMatches) > 1 {
		cnyValue, err := strconv.ParseFloat(cnyMatches[1], 64)
		if err == nil {
			currencyValues["CNY"] = cnyValue
		}
	}
	
	// Отправка результатов в канал
	for currency, value := range currencyValues {
		if value > 0 {
			ch <- CurrencyValue{currency: currency, value: value}
		} else {
			log.Printf("Не удалось получить значение для %s", currency)
		}
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
	order := [6]string{"BTC", "ETH", "SOL", "USD", "EUR", "CNY"}
	values := make(map[string]float64)
	var wg sync.WaitGroup
	ch := make(chan CurrencyValue, len(currencies))
	wg.Add(1)
	go getCryptoValues(ch, &wg)
	wg.Add(1)
	go getCurrencyPrices(ch, &wg)
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
