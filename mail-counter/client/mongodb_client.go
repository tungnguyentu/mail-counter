package client

import (
	"context"
	"sync"
	"time"

	mailcounter "github.com/Tungnt24/mail-counter/mail-counter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var clientInstance *mongo.Client
var clientInstanceError error
var mongoOnce sync.Once

var (
	CONNECTIONSTRING = mailcounter.Load().MongoUri
	DB               = mailcounter.Load().MongoDataBase
	MailLogs         = mailcounter.Load().MongoCollection
)

type MailLog struct {
	QueueId      string    `bson:"QueueId"`
	From         string    `bson:"From"`
	To           string    `bson:"To"`
	DomainFrom   string    `bson:"DomainFrom"`
	DomainTo     string    `bson:"DomainTo"`
	MessageId    string    `bson:"MessageId"`
	SenderSmtpIp string    `bson:"SenderSmtpIp"`
	Status       string    `bson:"Status"`
	SentAt       time.Time `bson:"SentAt"`
}

func ConvertToTimeMST(timeStr string) time.Time {
	layout := "2006-01-02 15:04:05 -0700 MST"
	timeParse, _ := time.Parse(layout, timeStr)
	return timeParse
}

func GetMongoClient() (*mongo.Client, error) {
	mongoOnce.Do(func() {
		clientOptions := options.Client().ApplyURI(CONNECTIONSTRING)
		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			clientInstanceError = err
		}
		err = client.Ping(context.TODO(), nil)
		if err != nil {
			clientInstanceError = err
		}
		clientInstance = client
	})
	return clientInstance, clientInstanceError
}
func CreateLog(task MailLog) error {
	client, err := GetMongoClient()
	if err != nil {
		return err
	}
	collection := client.Database(DB).Collection(MailLogs)
	_, err = collection.InsertOne(context.TODO(), task)
	if err != nil {
		return err
	}
	return nil
}

func GetManyLogs(key string, value string, fromDate time.Time, toDate time.Time) ([]MailLog, error) {
	filter := bson.M{
		"SentAt": bson.M{
			"$gte": primitive.NewDateTimeFromTime(fromDate),
			"$lt":  primitive.NewDateTimeFromTime(toDate),
		},
		key: value,
	}
	mailLog := []MailLog{}

	client, err := GetMongoClient()
	if err != nil {
		return mailLog, err
	}
	collection := client.Database(DB).Collection(MailLogs)

	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return mailLog, err
	}

	for cur.Next(context.TODO()) {
		t := MailLog{}
		err := cur.Decode(&t)
		if err != nil {
			return mailLog, err
		}
		mailLog = append(mailLog, t)
	}

	cur.Close(context.TODO())
	if len(mailLog) == 0 {
		return mailLog, mongo.ErrNoDocuments
	}
	return mailLog, nil
}

func UpdateLog(queueId string, to string, key string, value string) error {
	filter := bson.D{
		primitive.E{Key: "QueueId", Value: queueId},
	}
	updater := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: key, Value: value},
	}}}
	if key == "SentAt" {
		updater = bson.D{primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: key, Value: ConvertToTimeMST(value)},
		}}}
	}

	client, err := GetMongoClient()
	if err != nil {
		return err
	}
	collection := client.Database(DB).Collection(MailLogs)

	_, err = collection.UpdateOne(context.TODO(), filter, updater)
	if err != nil {
		return err
	}
	return nil
}

func GetLog(key string, value string) (MailLog, error) {
	result := MailLog{}
	filter := bson.D{primitive.E{Key: key, Value: value}}
	client, err := GetMongoClient()
	if err != nil {
		return result, err
	}
	collection := client.Database(DB).Collection(MailLogs)
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func GetLogs(key string, value string) (MailLog, error) {
	result := MailLog{}
	filter := bson.D{
		primitive.E{
			Key:   key,
			Value: value,
		},
	}
	client, err := GetMongoClient()
	if err != nil {
		return result, err
	}
	collection := client.Database(DB).Collection(MailLogs)
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}
