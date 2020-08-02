package astronomia

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/logging"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
)

// Bot runs as a Google Cloud Function.
func Bot(w http.ResponseWriter, r *http.Request) {

	// Always return 200 OK regardless of error. Side effect: Lost messages.
	defer w.WriteHeader(200)

	// Environment variables.
	token := os.Getenv("TOKEN")
	projectID := os.Getenv("GCP_PROJECT_ID")
	functionName := os.Getenv("FUNCTION_NAME")

	// Get context for GCP logging.
	ctx := context.Background()

	// We need to specify if this is a GCP project or not.
	if projectID == "" {
		log.Panic("[FATAL] Not a GCP project!")
	}
	if functionName == "" {
		log.Panic("[FATAL] Not a Google Cloud Function!")
	}

	// Initialize GCP client logger.
	logClient, err := logging.NewClient(ctx, projectID)
	if err != nil {
		log.Panicf("[FATAL] Stacktrace: %s!", err)
	}

	// Always close the client after function exited.
	defer logClient.Close()

	// Initialize logger from client.
	logger := logClient.Logger(
		"cloudfunctions.googleapis.com/cloud-functions",
		logging.CommonResource(&mrpb.MonitoredResource{
			Labels: map[string]string{
				"function_name": functionName, // GCF name.
				"project_id":    projectID,    // GCP project ID.
				"region":        "us-central1",
			},
			Type: "cloud_function",
		}),
		logging.CommonLabels(map[string]string{
			"execution_id": r.Header.Get("Function-Execution-Id"), // GCF execution ID.
		}),
	)

	// Initialize bot.
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Alert,
			Payload:  "Token invalid.",
		})
	}

	// Don't print out debug info from the library.
	bot.Debug = false

	logger.Log(logging.Entry{
		Severity: logging.Notice,
		Payload:  fmt.Sprintf("Bot %s started", bot.Self.UserName),
	})

	// Recover from panic, log the error.
	defer func() {
		err := recover()
		if err != nil {
			logger.Log(logging.Entry{
				Severity: logging.Critical,
				Payload:  fmt.Sprintf("Bot crashed. Stacktrace: %s", err),
			})
		}
	}()

	// Get new message, yay!
	update := bot.HandleUpdate(w, r)

	// Logs the message texts.
	if update.Message.Text != "" {
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  fmt.Sprintf("%s: %s", update.Message.From.UserName, update.Message.Text),
		})
	}

	// Check whether new message is a command or not.
	if update.Message.IsCommand() { // You thought it was not a command, but it was me, a command!

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "start":
			msg.Text = "Hello! Welcome to Astronomia Bot. Type /help to see all available commands."
		case "help":
			msg.Text = "Type /sayhi to say hi.\nType /status to get bot status.\nType /weather <location> to get current weather information.\nType /help to see all available commands."
		case "sayhi":
			// Check if the chat is in a group or not.
			if update.Message.Chat.IsGroup() {
				msg.Text = fmt.Sprintf("Hi %s! What a nice group chat, innit?", update.Message.Chat.UserName)
			} else {
				msg.Text = fmt.Sprintf("Hi %s %s! Have a good day!", update.Message.Chat.FirstName, update.Message.Chat.LastName)
			}
		case "status":
			msg.Text = "I'm ok. Thanks for asking anyway. \U0001F642"
		case "weather":
			if update.Message.CommandArguments() != "" {
				msg.Text, err = GetWeather(update.Message.Chat.FirstName, update.Message.Chat.LastName, update.Message.Text)

				// Oops, weather not found for this location, or some error happened.
				if err != nil {
					logger.Log(logging.Entry{
						Severity: logging.Error,
						Payload:  fmt.Sprintf("Error: %s", err),
					})

					// Say sorry and spit out the error, even when it panicked lol.
					msg.Text = "Thousand apologize!. It appears that " + msg.Text + ". Please try another location!"
				}
			}
		default:
			msg.Text = "I don't know that command."
		}

		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  fmt.Sprintf("%s: %s", bot.Self.UserName, msg.Text),
		})

		// Send the reply.
		bot.Send(msg)
	}
	return
}
