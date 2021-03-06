package main

import (
	"errors"
	"github.com/Financial-Times/go-logger"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/dchest/uniuri"
)

const tidValidRegexp = "(tid|SYNTHETIC-REQ-MON)[a-zA-Z0-9_-]*$"

func (bridge BridgeApp) forwardMsg(msg queueConsumer.Message) {
	tid, err := extractTID(msg.Headers)
	if err != nil {
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		logger.NewEntry(tid).Info("Couldn't extract transaction id, due to %s. TID was generated.", err.Error())
	}
	msg.Headers["X-Request-Id"] = tid
	err = bridge.producerInstance.SendMessage("", queueProducer.Message{Headers: msg.Headers, Body: msg.Body})
	if err != nil {
		logger.NewMonitoringEntry("Forwarding", tid, "").Error("Error happened during message forwarding: " + err.Error())
	} else {
		logger.NewMonitoringEntry("Forwarding", tid, "").Info("Message has been forwarded")
	}
}

func extractTID(headers map[string]string) (string, error) {
	header := headers["X-Request-Id"]
	if header == "" {
		return "", errors.New("X-Request-Id header could not be found.")
	}
	return header, nil
}
