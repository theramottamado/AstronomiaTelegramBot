package astronomia

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
)

type LinkedID struct {
	UserID  int
	GroupID int64
}

var (
	unames map[LinkedID]bool
)

func init() {
	unames = map[LinkedID]bool{}
}

func AstronomiaBot(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if os.Getenv("GCLOUD_PROJECT") == "" {
		log.Panic("[FATAL] Not a gcloud project!")
	}
	logClient, err := logging.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	if err != nil {
		log.Panic("[FATAL] Not a gcloud project!")
	}
	defer logClient.Close()
	if os.Getenv("FUNCTION_NAME") == "" {
		log.Panic("[FATAL] Not a cloud function!")
	}
	logger := logClient.Logger(
		"cloudfunctions.googleapis.com/cloud-functions",
		logging.CommonResource(&mrpb.MonitoredResource{
			Labels: map[string]string{
				"function_name": os.Getenv("FUNCTION_NAME"),
				"project_id":    os.Getenv("GCP_PROJECT"),
				"region":        "us-central1",
			},
			Type: "cloud_function",
		}),
		logging.CommonLabels(map[string]string{
			"execution_id": r.Header.Get("Function-Execution-Id"),
		}),
	)

	token := os.Getenv("TOKEN")
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		// log.Panic("[FATAL] No webhook URL specified!")
		logger.Log(logging.Entry{
			Severity: logging.Alert,
			Payload:  "No webhook URL specified!",
		})
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		// log.Panic("[FATAL] Token invalid.")
		logger.Log(logging.Entry{
			Severity: logging.Alert,
			Payload:  "Token invalid.",
		})
	}

	bot.Debug = false

	// log.Printf("[INFO] Authorized on account: %s", bot.Self.UserName)
	logger.Log(logging.Entry{
		Severity: logging.Debug,
		Payload:  fmt.Sprintf("Authorized on account: %s", bot.Self.UserName),
	})

	_, err = bot.SetWebhook(tgbotapi.NewWebhook(webhookURL + "?" + bot.Token))
	if err != nil {
		// log.Fatalf("[FATAL] Bot crashed. Stacktrace: %s", err)
		logger.Log(logging.Entry{
			Severity: logging.Critical,
			Payload:  fmt.Sprintf("Bot crashed. Stacktrace: %s", err),
		})
	}
	_, err = bot.GetWebhookInfo()
	if err != nil {
		// log.Fatalf("[FATAL] Bot crashed. Stacktrace: %s", err)
		logger.Log(logging.Entry{
			Severity: logging.Critical,
			Payload:  fmt.Sprintf("Bot crashed. Stacktrace: %s", err),
		})
	}
	defer func() {
		err := recover()
		if err != nil {
			// log.Printf("[FATAL] Bot crashed. Stacktrace: %s", err)
			logger.Log(logging.Entry{
				Severity: logging.Critical,
				Payload:  fmt.Sprintf("Bot crashed. Stacktrace: %s", err),
			})
		}
		w.WriteHeader(200)
	}()

	update := bot.HandleUpdate(w, r)

	if update.Message.Text != "" {
		// log.Printf("[INFO] %s: %s", update.Message.From.UserName, update.Message.Text)
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  fmt.Sprintf("%s: %s", update.Message.From.UserName, update.Message.Text),
		})
	}
	// log.Printf("[DEBUG] Length of map is: %d", len(unames))

	if !update.Message.IsCommand() {
		// log.Printf("[DEBUG] User ID: %d", update.Message.From.ID)
		// log.Printf("[DEBUG] Chat ID: %d", update.Message.Chat.ID)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		if _, ok := unames[LinkedID{UserID: update.Message.From.ID, GroupID: update.Message.Chat.ID}]; ok {
			delete(unames, LinkedID{update.Message.From.ID, update.Message.Chat.ID})
			msg.Text, err = GetWeather(update.Message.Chat.FirstName, update.Message.Chat.LastName, update.Message.Text)
			if err != nil {
				// log.Printf("[ERROR] %s", err)
				logger.Log(logging.Entry{
					Severity: logging.Error,
					Payload:  fmt.Sprintf("Error: %s", err),
				})
				msg.Text = "It appears that " + fmt.Sprintf("error %s", err) + ", try another location!"
				// log.Printf("[INFO] %s: %s", bot.Self.UserName, msg.Text)
				logger.Log(logging.Entry{
					Severity: logging.Info,
					Payload:  fmt.Sprintf("%s: %s", bot.Self.UserName, msg.Text),
				})
				bot.Send(msg)
				return
			}
			// log.Printf("[INFO] %s: %s", bot.Self.UserName, msg.Text)
			logger.Log(logging.Entry{
				Severity: logging.Info,
				Payload:  fmt.Sprintf("%s: %s", bot.Self.UserName, msg.Text),
			})
			bot.Send(msg)
		}
	} else if update.Message.IsCommand() {
		delete(unames, LinkedID{UserID: update.Message.From.ID, GroupID: update.Message.Chat.ID})
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		switch update.Message.Command() {
		case "start":
			msg.Text = "Hello! Welcome to Astronomia Bot. Type /help to see all available commands."
		case "help":
			msg.Text = "Type /sayhi to say hi.\nType /status to get bot status.\nType /weather to get current weather information.\nType /help to see all available commands."
		case "sayhi":
			if update.Message.Chat.IsGroup() {
				msg.Text = fmt.Sprintf("Hi %s!", update.Message.Chat.UserName)
			} else {
				msg.Text = fmt.Sprintf("Hi %s %s! Have a good day!", update.Message.Chat.FirstName, update.Message.Chat.LastName)
			}
		case "status":
			msg.Text = "I'm ok."
		case "weather":
			unames[LinkedID{update.Message.From.ID, update.Message.Chat.ID}] = true
			msg.Text = "What is the address you want to know the weather of?"
		default:
			msg.Text = "I don't know that command"
		}
		// log.Printf("[INFO] %s: %s", bot.Self.UserName, msg.Text)
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  fmt.Sprintf("%s: %s", bot.Self.UserName, msg.Text),
		})
		bot.Send(msg)
	}
	return
}
