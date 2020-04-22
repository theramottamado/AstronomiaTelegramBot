package main

import (
	"os"

	"github.com/AstronomiaDev/AstronomiaTelegramBot/cmd"
)

func main() {
	token := os.Getenv("TOKEN")
	cmd.Bot(token)
}
