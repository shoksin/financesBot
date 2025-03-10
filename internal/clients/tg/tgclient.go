package tg

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(tgUpdate tgbotapi.Update, c *Client, msgModel *messages.Model)

func (f HandlerFunc) RunFunc(tgUpdate tgbotapi.Update, c *Client, msgModel *messages.Model) {
	f(tgUpdate, c, msgModel)
}
