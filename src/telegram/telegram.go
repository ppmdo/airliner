package telegram

import (
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "io"
    "log"
)

const chatID = int64(-470314890)

type Bot = tgbotapi.BotAPI

func InitBot() (*tgbotapi.BotAPI, error) {
    return tgbotapi.NewBotAPI("1815941771:AAHDzmF2KLNp45XlD14_Bi0iXmocekf2gaE")
}

func handleError(err error) {
    if err != nil {
        log.Panic(err)
    }
}

func SendImage(bot *tgbotapi.BotAPI, filename string, reader io.Reader) {
    file := tgbotapi.FileReader{
        Name: filename,
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