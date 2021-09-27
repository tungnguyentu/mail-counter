package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	mailcounter "github.com/Tungnt24/mail-counter/mail-counter"
)

func SendTele(message []byte) []byte {
	cfg := mailcounter.Load()
	telegramApi := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.TelegramBotToken)
	resp, err := http.Post(telegramApi, "application/json", bytes.NewBuffer(message))
	if err != nil {
		fmt.Print(err)
		return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}
