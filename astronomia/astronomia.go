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

// LinkedID is a struct to save current UserID and ChatID of user.
type LinkedID struct {
	FromID int
	ChatID int64
}

// Global variables.
var (
	unames map[LinkedID]bool
)

// This should run on the start of deployment.
func init() {
	unames = map[LinkedID]bool{}
}

// AstronomiaBot runs as a Google Cloud Function.
func AstronomiaBot(w http.ResponseWriter, r *http.Request) {

	// Always return 200 OK regardless of error. Side effect: Lost messages.
	defer w.WriteHeader(200)

	// Environment variables.
	token := os.Getenv("TOKEN")
	// webhookURL := os.Getenv("WEBHOOK_URL") // Not used
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
		"cloudfunctions.googleapis.com/cloud-functions", // This is the Log ID.
		logging.CommonResource(&mrpb.MonitoredResource{
			Labels: map[string]string{
				"function_name": functionName, // GCF name.
				"project_id":    projectID,    // GCP project ID.
				"region":        "us-central1",
			},
			Type: "cloud_function", // Because this is a GCF.
		}),
		logging.CommonLabels(map[string]string{
			"execution_id": r.Header.Get("Function-Execution-Id"), // GCF execution ID.
		}),
	)

	// Initialize bot.
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		// Logging is fun!
		logger.Log(logging.Entry{
			Severity: logging.Alert,
			Payload:  "Token invalid.",
		})
	}

	// Don't print out debug info from the library.
	bot.Debug = true

	// Logging is fun!
	logger.Log(logging.Entry{
		Severity: logging.Notice,
		Payload:  fmt.Sprintf("Bot %s started", bot.Self.UserName),
	})

	/*** These blocks should not run.

	// These following lines are kinda costly though, we should just not check and set webhook info from here.
	// So I decided to comment out these part where we check and set webhook info.

	// Check if webhook URL is specified.
	if webhookURL == "" {
		logger.Log(logging.Entry{
			Severity: logging.Alert,
			Payload:  "No webhook URL specified!",
		})
	}

	// Set webhook config.
	webhookConfig := tgbotapi.NewWebhook(webhookURL + "?" + bot.Token)

	// Get webhook info. This is to prevent us set the webhook info for each invocation.
	webhookInfo, err := bot.GetWebhookInfo()
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Critical,
			Payload:  fmt.Sprintf("Bot crashed. Stacktrace: %s", err),
		})
	}

	// Set webhook url if it's not set.
	if webhookInfo.URL != webhookConfig.URL.String() {
		_, err = bot.SetWebhook(webhookConfig)
		if err != nil {
			logger.Log(logging.Entry{
				Severity: logging.Critical,
				Payload:  fmt.Sprintf("Bot crashed. Stacktrace: %s", err),
			})
		}
	}

	***/

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
	if !update.Message.IsCommand() { // If not a command, then ...
		// Initialize the reply
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		// Check if this is a response of weather command.
		if _, ok := unames[LinkedID{FromID: update.Message.From.ID, ChatID: update.Message.Chat.ID}]; ok {
			// This is a response. Delete the user from the response queue.
			delete(unames, LinkedID{update.Message.From.ID, update.Message.Chat.ID})

			// Fun part: Get the current weather condition.
			msg.Text, err = GetWeather(update.Message.Chat.FirstName, update.Message.Chat.LastName, update.Message.Text)
			if err != nil { // Oops, weather not found for this location, or some error happened.
				logger.Log(logging.Entry{
					Severity: logging.Error,
					Payload:  fmt.Sprintf("Error: %s", err),
				})

				// Say sorry and spit out the error, even when it panicked lol.
				msg.Text = "Thousand apologize!. It appears that " + msg.Text + ". Please try another location!"
				logger.Log(logging.Entry{
					Severity: logging.Info,
					Payload:  fmt.Sprintf("%s: %s", bot.Self.UserName, msg.Text),
				})

				// Send the reply.
				bot.Send(msg)
				return
			}

			// Yay, we get the weather!
			logger.Log(logging.Entry{
				Severity: logging.Info,
				Payload:  fmt.Sprintf("%s: %s", bot.Self.UserName, msg.Text),
			})

			// Send the reply of current weather.
			msg.ParseMode = "HTML"
			bot.Send(msg)
		}
	} else if update.Message.IsCommand() { // You thought it was not a command, but it was me, a command!
		// Bah, so you decided not to check the weather!
		delete(unames, LinkedID{FromID: update.Message.From.ID, ChatID: update.Message.Chat.ID})
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		// What do you need me to do, my master?
		switch update.Message.Command() {
		case "start": // Lame, but Gene asked for this and opened an issue!
			msg.Text = "Hello! Welcome to Astronomia Bot. Type /help to see all available commands."
		case "help": // You really need help? Pathetic.
			msg.Text = "Type /sayhi to say hi.\nType /status to get bot status.\nType /weather to get current weather information.\nType /help to see all available commands."
		case "sayhi": // You want me to say hi to you?
			// Check if the chat is in a group or not.
			if update.Message.Chat.IsGroup() {
				msg.Text = fmt.Sprintf("Hi %s! What a nice group chat, innit?", update.Message.Chat.UserName)
			} else {
				msg.Text = fmt.Sprintf("Hi %s %s! Have a good day!", update.Message.Chat.FirstName, update.Message.Chat.LastName)
			}
		case "status": // Do you really want to know my status? ;)
			msg.Text = "I'm ok. Thanks for asking anyway. \U0001F642"
		case "weather": // You asking for weather report?
			unames[LinkedID{update.Message.From.ID, update.Message.Chat.ID}] = true
			msg.Text = "What is the address you want to know the weather of?"
		default: // Sorry I'm still dumb.
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
