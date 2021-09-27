package main

import (
	"encoding/json"
	"fmt"

	mailcounter "github.com/Tungnt24/mail-counter/mail-counter"
	"github.com/Tungnt24/mail-counter/mail-counter/client"
	"github.com/Tungnt24/mail-counter/mail-counter/utils"
	"github.com/jasonlvhit/gocron"
	"github.com/sirupsen/logrus"
)

type sendMessageReqBody struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func Task(duration int) {
	cfg := mailcounter.Load()
	message := `
		Counter_Time: %v
		per_minute: %d
	`
	timeFrom, counter := utils.Counter(duration)
	message = fmt.Sprintf(message, timeFrom, counter)
	logrus.Info(message)
	reqBody := &sendMessageReqBody{
		ChatID: cfg.TelegramChatId,
		Text:   message,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		logrus.Error(err)
	}
	if counter > 10 {
		logrus.Info("Sending to telegram.....")
		client.SendTele(reqBytes)
		logrus.Info("Done")
	}
}

func main() {
	utils.InitLog()
	gocron.Every(1).Minute().Do(Task, 1)
	<-gocron.Start()
}
