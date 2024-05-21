package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"

	"errors"
	"os"
	"strconv"
)

var chatID = int64(0)

type Bot = tgbotapi.BotAPI

func InitBot() (*tgbotapi.BotAPI, error) {
	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		return nil, errors.New("TELEGRAM_CHAT_ID is not set")
	}

	v, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return nil, err
	}
	chatID = int64(v)

	tokenStr := os.Getenv("TELEGRAM_BOT_TOKEN")
	if tokenStr == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is not set")
	}

	return tgbotapi.NewBotAPI(tokenStr)
}

func handleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func SendImage(bot *tgbotapi.BotAPI, filename string, reader io.Reader) {
	file := tgbotapi.FileReader{
		Name:   filename,
		Reader: reader,
	}
	imgMsg := tgbotapi.NewPhoto(chatID, file)
	_, err := bot.Send(imgMsg)
	handleError(err)
}

func SendMessage(bot *tgbotapi.BotAPI, msgText string) {
	_, err := bot.Send(
		tgbotapi.NewMessage(chatID, msgText),
	)
	handleError(err)
}
