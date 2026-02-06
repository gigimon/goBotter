package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type updatesDebugHTTPClient struct {
	base *http.Client
}

func (c *updatesDebugHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil && strings.HasSuffix(req.URL.Path, "/getUpdates") && req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			log.Printf("getUpdates payload: %s", string(body))
			req.Body = io.NopCloser(bytes.NewReader(body))
		} else {
			log.Println("Can't read getUpdates request body")
			log.Println(err)
		}
	}
	return c.base.Do(req)
}

func main() {
	log.Println("Start application")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handleAllMessages),
		bot.WithAllowedUpdates(bot.AllowedUpdates{
			models.AllowedUpdateMessage,
			models.AllowedUpdateEditedMessage,
			models.AllowedUpdateCallbackQuery,
			models.AllowedUpdateMessageReaction,
			models.AllowedUpdateMessageReactionCount,
		}),
		bot.WithHTTPClient(61*time.Second, &updatesDebugHTTPClient{
			base: &http.Client{Timeout: 61 * time.Second},
		}),
	}

	token := os.Getenv("TELEGRAM_API_TOKEN")

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
	goBotter.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update != nil && update.MessageReaction != nil
	}, handleReactionUpdate)
	goBotter.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update != nil && update.MessageReactionCount != nil
	}, handleReactionCountUpdate)

	log.Println("Start bot")
	goBotter.Start(ctx)
}
