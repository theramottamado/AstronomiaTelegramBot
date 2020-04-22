package astronomia

import (
	"os"

	"github.com/AstronomiaDev/AstronomiaTelegramBot/cmd"
)

func Astronomia() {
	token := os.Getenv("TOKEN")
	cmd.Bot(token)
}
