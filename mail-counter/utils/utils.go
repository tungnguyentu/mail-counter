package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Tungnt24/mail-counter/mail-counter/client"
	"github.com/sirupsen/logrus"
)

func FilterLog(message string) bool {
	if strings.Contains(message, "Passed CLEAN") &&
		strings.Contains(message, "Message-ID") ||
		strings.Contains(message, "postfix/lmtp") &&
			strings.Contains(message, "status=sent") {
		return true
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

func ExtractEmail(email string) (string, string) {
	atSign := strings.Index(email, "@")
	return email[:atSign], email[atSign+1:]
}

func CollectField(rawMessageStr string) (client.MailLog, error) {
	mailLog := client.MailLog{}
	mapping := Dump(rawMessageStr)
	rawMessage := fmt.Sprintf("%v\n", mapping["message"])
	logrus.Info("\nMessage: ", rawMessage)
	timestamp := fmt.Sprintf("%v\n", mapping["@timestamp"])
	index := strings.Index(rawMessage, "]:")
	message := strings.TrimSpace(rawMessage[index+2:])
	fields := strings.Split(message, " ")
	re := regexp.MustCompile(`=`)
	if strings.Contains(rawMessage, "amavis") {
		fields = strings.Split(message, ",")
		re = regexp.MustCompile(`:`)
	}
	queueId := strings.Trim(strings.Replace(string(fields[0]), ":", "", 1), "")
	for _, field := range fields {
		items := re.Split(field, 2)
		if len(items) <= 1 {
			continue
		}
		rawKey, rawValue := strings.Trim(items[0], " "), strings.Trim(items[1], " ")
		key := strings.Title(rawKey)
		value := strings.Trim(strings.Replace(rawValue, ",", " ", 1), " ")
		switch {
		case key == "Queued_as":
			queueId = value
		case key == "Message-ID":
			mailLog.MessageId = value
			mailLog.SentAt = ConvertToTimeUTC(strings.Trim(timestamp, "\n"))
		case strings.Contains(key, "["):
			mailLog.SenderSmtpIp = key
			i := strings.Index(value, "<")
			emails := strings.Split(value[i:], "->")
			emailFrom := strings.Replace(strings.Replace(emails[0], "<", "", 1), ">", "", 1)
			emailTo := strings.Replace(strings.Replace(emails[1], "<", "", 1), ">", "", 1)
			if emailFrom == " " || emailTo == " " {
				continue
			}
			_, domainFrom := ExtractEmail(emailFrom)
			_, domainTo := ExtractEmail(emailTo)
			mailLog.From = strings.Trim(emailFrom, " ")
			mailLog.To = strings.Trim(emailTo, " ")
			mailLog.DomainFrom = domainFrom
			mailLog.DomainTo = domainTo
		case key == "Status":
			mailLog.Status = value
		case key == "To":
			mailLog.To = strings.Trim(strings.Replace(strings.Replace(value, "<", "", 1), ">", "", 1), " ")
		}
	}
	mailLog.QueueId = queueId
	return mailLog, nil
}

func AggregateLog(mailLog client.MailLog) {
	v := reflect.ValueOf(mailLog)
	typeOfS := v.Type()
	result, _ := client.GetLogs("QueueId", mailLog.QueueId)
	if result != (client.MailLog{}) {
		for i := 1; i < v.NumField(); i++ {
			key := typeOfS.Field(i).Name
			value := fmt.Sprintf("%v", v.Field(i).Interface())
			if value == "" || value == "0001-01-01 00:00:00 +0000 UTC" {
				continue
			}
			client.UpdateLog(mailLog.QueueId, mailLog.To, key, value)
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

func GetTime(duration int) (time.Time, time.Time) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Now().In(loc)
	nowUtcStr := now.Format(time.RFC3339)
	thenStr := now.Add(time.Duration(-duration) * time.Minute).In(loc).Format(time.RFC3339)
	toDatetime := ConvertToTimeUTC(nowUtcStr)
	fromDatetime := ConvertToTimeUTC(thenStr)
	return fromDatetime, toDatetime
}
