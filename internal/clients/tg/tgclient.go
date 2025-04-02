package tg

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shoksin/financesBot/internal/logger"
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
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.client.GetUpdatesChan(u)

	logger.Info("Start listenin for tg messages")

	for update := range updates {
		// Функция обработки сообщений (обернутая в middleware).
		c.handlerProcessingFunc.RunFunc(update, c, msgModel)
		//вместо ProcessingMessages(update, c, msgModel)
	}
}

// ProcessingMessages функция обработки сообщений.
func ProcessingMessages(tgUpdate tgbotapi.Update, c *Client, msgModel *messages.Model) {
	if tgUpdate.Message != nil {
		//Пользователь написал текстовое сообщение
		logger.Info(fmt.Sprintf("[%s][%v] %s", tgUpdate.Message.From.UserName, tgUpdate.Message.From.ID, tgUpdate.Message.Text))
		err := msgModel.IncomingMessage(messages.Message{
			Text:            tgUpdate.Message.Text,
			UserID:          tgUpdate.Message.From.ID,
			UserName:        tgUpdate.Message.From.UserName,
			UserDisplayName: strings.TrimSpace(tgUpdate.CallbackQuery.From.FirstName + " " + tgUpdate.CallbackQuery.From.LastName),
			IsCallback:      true,
			CallbackMsgID:   tgUpdate.CallbackQuery.ID,
		})

		if err != nil {
			logger.Error("error processing message:", "err", err)
		}
	} else if tgUpdate.CallbackQuery != nil {
		// Пользователь нажал кнопку.
		logger.Info(fmt.Sprintf("[%s][%v] Callback: %s", tgUpdate.CallbackQuery.From.UserName, tgUpdate.CallbackQuery.From.ID, tgUpdate.CallbackQuery.Data))
		callback := tgbotapi.NewCallback(tgUpdate.CallbackQuery.ID, tgUpdate.CallbackQuery.Data)
		if _, err := c.client.Request(callback); err != nil {
			logger.Error("Error Request callback:", "err", err)
		}
		if err := deleteInlineButtons(c, tgUpdate.CallbackQuery.From.ID, tgUpdate.CallbackQuery.Message.MessageID, tgUpdate.CallbackQuery.Message.Text); err != nil {
			logger.Error("Error deleting buttons", "err", err)
		}
		err := msgModel.IncomingMessage(messages.Message{})
		if err != nil {
			logger.Error("error processing message from callback:", "err", err)
		}
	}
}

func deleteInlineButtons(c *Client, userID int64, msgID int, sourceText string) error {
	msg := tgbotapi.NewEditMessageText(userID, msgID, sourceText)
	_, err := c.client.Send(msg)
	if err != nil {
		logger.Error("Error sending message", "err", err)
		return fmt.Errorf("client.Send remove inline-buttons error: %v", err)
	}
	return nil
}
