package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Shopify/sarama"
	mailcounter "github.com/Tungnt24/mail-counter/mail_counter"
	"github.com/Tungnt24/mail-counter/mail_counter/utils"
	"github.com/sirupsen/logrus"
)

type Consumer struct {
	ready chan bool
}

func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	close(consumer.ready)
	return nil
}

func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for rawMessage := range claim.Messages() {
		rawMessageStr := string(rawMessage.Value)
		is_legal_message := utils.FilterLog(rawMessageStr)
		if is_legal_message {
			log, _ := utils.CollectField(rawMessageStr)
			utils.AggregateLog(log)
		}

		session.MarkMessage(rawMessage, "")
	}
	return nil
}

func ConnectConsumer(brokersUrl []string, groupId string) (sarama.ConsumerGroup, error) {
	logrus.Info("Starting a new Sarama consumer")
	sarama.Logger = log.New(os.Stdout, "[bounce_mail] ", log.LstdFlags)

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	client, err := sarama.NewConsumerGroup(brokersUrl, groupId, config)
	if err != nil {
		logrus.Error("Error creating consumer group client: %v", err)
	}
	return client, nil
}

func Worker() {
	cfg := mailcounter.Load()
	consumer := Consumer{
		ready: make(chan bool),
	}
	ctx, cancel := context.WithCancel(context.Background())
	broker := cfg.KafkaBroker
	topics := cfg.KafkaTopic
	groupId := cfg.KafkaConsumerGroup
	client, err := ConnectConsumer(broker, groupId)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := client.Consume(ctx, strings.Split(topics, ","), &consumer); err != nil {
				logrus.Error("Error from consumer: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready
	logrus.Info("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		logrus.Info("terminating: context cancelled")
	case <-sigterm:
		logrus.Info("terminating: via signal")
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		logrus.Error("Error closing client: %v", err)
	}
}

func main() {
	utils.InitLog()
	Worker()
}
