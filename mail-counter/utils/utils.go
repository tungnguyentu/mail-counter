package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Tungnt24/mail-counter/mail_counter/client"
	"github.com/sirupsen/logrus"
)

func FilterLog(message string) bool {
	if strings.Contains(message, "capt-se") {
		if strings.Contains(message, "postfix/qmgr") &&
			strings.Contains(message, "from=") ||
			strings.Contains(message, "postfix/cleanup") &&
				strings.Contains(message, "message-id") ||
			strings.Contains(message, "postfix/smtp") &&
				strings.Contains(message, "status=bounced") {
			return true
		}
	}

	return false
}

func Dump(message string) map[string]interface{} {
	mapping := make(map[string]interface{})
	err := json.Unmarshal([]byte(message), &mapping)

	if err != nil {
		panic(err)
	}
	return mapping
}

func ConvertToTimeUTC(timeStr string) time.Time {
	layout := "2006-01-02T15:04:05Z"
	timeParse, _ := time.Parse(layout, timeStr)
	return timeParse
}

func CollectField(rawMessageStr string) (client.MailLog, error) {
	mail_log := client.MailLog{}
	mapping := Dump(rawMessageStr)
	rawMessage := fmt.Sprintf("%v\n", mapping["message"])
	logrus.Info("Message: ", rawMessage)
	timestamp := fmt.Sprintf("%v\n", mapping["@timestamp"])
	index := strings.Index(rawMessage, "]:")
	statusMessageIndex := strings.Index(rawMessage, "(")
	if statusMessageIndex == -1 {
		statusMessageIndex = len(rawMessage) - 1
	}
	message := strings.TrimSpace(rawMessage[index+2 : statusMessageIndex])
	fields := strings.Split(message, " ")
	re := regexp.MustCompile(`=`)
	queueId := strings.Trim(strings.Replace(string(fields[0]), ":", "", 1), "")
	for _, field := range fields[1:] {
		items := re.Split(field, 2)
		if len(items) <= 1 {
			continue
		}
		raw_key, raw_value := strings.Trim(items[0], " "), strings.Trim(items[1], " ")
		key := strings.Title(raw_key)
		value := strings.Replace(raw_value, ",", " ", 1)
		switch key {
		case "From":
			mail_log.From = value
		case "To":
			mail_log.To = value
		case "Message-Id":
			mail_log.MessageId = value
		case "Relay":
			open_char := strings.Index(value, "[")
			close_char := strings.Index(value, "]")
			if open_char == -1 || close_char == -1 {
				continue
			}
			mail_log.RecipientSmtpDomain = value[:open_char]
			mail_log.RecipientSmtpIp = value[open_char+1 : close_char]
		case "Status":
			mail_log.SentAt = ConvertToTimeUTC(strings.Trim(timestamp, "\n"))
			status_message := rawMessage[statusMessageIndex:]
			mail_log.Status = value
			mail_log.Message = status_message
		}
	}
	mail_log.QueueId = queueId
	return mail_log, nil
}

func AggregateLog(mailLog client.MailLog) {
	v := reflect.ValueOf(mailLog)
	typeOfS := v.Type()
	result, _ := client.GetLogByQueueId(mailLog.QueueId)
	if result != (client.MailLog{}) {
		for i := 1; i < v.NumField(); i++ {
			key := strings.ToLower(typeOfS.Field(i).Name)
			value := fmt.Sprintf("%v", v.Field(i).Interface())
			if value == "" || value == "0001-01-01 00:00:00 +0000 UTC" {
				continue
			}
			client.UpdateLog(mailLog.QueueId, key, value)
		}
	} else {
		client.CreateLog(mailLog)
	}
}

func DetectSpam(message string) bool {
	spamPattern := `^.*(\breputation\b|\bspam\b|\bspamhaus\b|\blisted\b|\bblock\b|\bblocked\b|\bsecurity\b|\bblacklisted\b|\bphish\b|\bphishing\b|\bvirus\b|\brejected\b|\bblacklisted\b|\bblacklist\b).*($|[^\w])`
	message = strings.ToLower(message)
	match, _ := regexp.MatchString(spamPattern, message)
	return match
}

func GetBounceMail(fromDatetime time.Time, toDatetime time.Time) []client.MailLog {
	key := "status"
	value := "bounced"
	result, _ := client.GetManyLogs(key, value, fromDatetime, toDatetime)
	return result
}

func GetTime(duration int) (time.Time, time.Time) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Now().In(loc)
	nowUtcStr := now.Format(time.RFC3339)
	thenStr := now.Add(time.Duration(-duration) * time.Minute).In(loc).Format(time.RFC3339)
	toDatetime := ConvertToTimeUTC(nowUtcStr)
	fromDatetime := ConvertToTimeUTC(thenStr)
	return fromDatetime, toDatetime
}

func Counter(duration int) (time.Time, int) {
	from, to := GetTime(duration)
	counter := 0
	result := GetBounceMail(from, to)
	for _, value := range result {
		isSpam := DetectSpam(value.Message)
		if isSpam {
			counter += 1
		}
	}
	return from, counter
}
