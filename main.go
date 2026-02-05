package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
)

func main() {
	log.Println("Start application")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handleAllMessages),
	}

	debug := os.Getenv("DEBUG")
	if len(debug) != 0 {
		opts = append(opts, bot.WithDebug())
	}

	goBotter, err := bot.New(os.Getenv("TELEGRAM_API_TOKEN"), opts...)
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

	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топдень", bot.MatchTypePrefix, handleDayTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топ", bot.MatchTypePrefix, handleTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!моястата", bot.MatchTypeExact, handleMyStat)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топреакдень", bot.MatchTypeExact, handleReactionDayTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!топреак", bot.MatchTypeExact, handleReactionTop)
	goBotter.RegisterHandler(bot.HandlerTypeMessageText, "!мояреак", bot.MatchTypeExact, handleMyReaction)

	log.Println("Start bot")
	goBotter.Start(ctx)
}
