package tg

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shoksin/financesBot/internal/models/messages"
)

type HandlerFunc func(tgUpdate tgbotapi.Update, c *Client, msgModel *messages.Model)

func (f HandlerFunc) RunFunc(tgUpdate tgbotapi.Update, c *Client, msgModel *messages.Model) {
	f(tgUpdate, c, msgModel)
}

type Client struct {
	client                *tgbotapi.BotAPI
	handlerProcessingFunc HandlerFunc // Функция обработки входящих сообщений.
}

type TokenGetter interface {
	Token() string
}

func New(tokenGetter TokenGetter, handlerProcessingFunc HandlerFunc) (*Client, error) {
	client, err := tgbotapi.NewBotAPI(tokenGetter.Token())
	if err != nil {
		return nil, fmt.Errorf("error NewBotAPI: %v", err)
	}

	return &Client{
		client:                client,
		handlerProcessingFunc: handlerProcessingFunc,
	}, nil
}

func (c *Client) SendMessage(userID int64, text string) error {
	msg := tgbotapi.NewMessage(userID, text) //Создаём конфиг сообщения
	msg.ParseMode = "markdown"
	_, err := c.client.Send(msg) //отправляем сообщение в telegram
	if err != nil {
		return fmt.Errorf("error sending message client.Send: %v", err)
	}
	return nil
}

func (c *Client) ListenUpdates(msgModel *messages.Model) {

}
