package cmd

import (
	"log"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func Bot(token string) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	uname := ""

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.IsCommand() != true {
			if uname != "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
				resp := update.Message.Text
				log.Println(resp)
				msg.Text = GetWeather(resp)
				bot.Send(msg)
				uname = ""
			}
		} else if update.Message.IsCommand() {
			uname = ""
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "help":
				msg.Text = "Type /sayhi or /status or / to see all available command"
			case "sayhi":
				msg.Text = "Hi :)"
			case "status":
				msg.Text = "I'm ok."
			case "weather":
				uname = update.Message.From.UserName
				msg.Text = "Which city?"
			default:
				msg.Text = "I don't know that command"
			}
			bot.Send(msg)
		}
	}
}
