package cmd

import (
	"container/list"
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

	weatherChan := make(chan string)

	updates, err := bot.GetUpdatesChan(u)

	unames := list.New()
	messages := list.New()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.IsCommand() != true {
			log.Println(unames.Len())
			if unames.Len() > 0 {
				if update.Message.From.UserName == unames.Front().Value.(string) {
					messages.PushFront(update.Message.Text)
					if messages.Len() == unames.Len() {
						for unames.Len() > 0 {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
							go getWeather(messages.Front().Value.(string), weatherChan)
							msg.Text = <-weatherChan
							bot.Send(msg)
							unames.Remove(unames.Front())
							messages.Remove(messages.Front())
						}
					}
				} else if unames.Len() > 1 && update.Message.From.UserName == unames.Back().Value.(string) {
					messages.PushBack(update.Message.Text)
				}
			}
		} else if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			uname := update.Message.From.UserName
			if unames.Len() > 0 && uname == unames.Front().Value.(string) {
				unames.Remove(unames.Front())
			}
			switch update.Message.Command() {
			case "help":
				msg.Text = "Type /sayhi or /status or / to see all available command"
			case "sayhi":
				msg.Text = "Hi :)"
			case "status":
				msg.Text = "I'm ok."
			case "weather":
				if unames.Len() < 1 {
					unames.PushBack(uname)
				} else if uname != unames.Front().Value.(string) {
					unames.PushBack(uname)
				}
				msg.Text = "What is the address you want to know the weather for?"
			default:
				msg.Text = "I don't know that command"
			}
			bot.Send(msg)
		}
	}
}

func getWeather(address string, weatherChan chan string) {
	msg := GetWeather(address)
	weatherChan <- msg
}
