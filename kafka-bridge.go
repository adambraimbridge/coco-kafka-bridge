/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Taken from: https://github.com/stealthly/go_kafka_client/blob/master/consumers/consumers.go
 *
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	fthealth "github.com/Financial-Times/go-fthealth"
	"github.com/dchest/uniuri"
	kafkaClient "github.com/stealthly/go_kafka_client"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig *kafkaClient.ConsumerConfig
	topic          string
	httpClient     *http.Client
	httpHost       string
}

func newBridgeApp(confPath string) (*BridgeApp, int) {
	consumerConfig, host, topic, numConsumers := ResolveConfig(confPath)
	bridgeApp := &BridgeApp{
		consumerConfig: consumerConfig,
		topic:          topic,
		httpClient:     &http.Client{},
		httpHost:       strings.Trim(host, "/"),
	}
	return bridgeApp, numConsumers
}

func (bridge BridgeApp) startNewConsumer() *kafkaClient.Consumer {
	consumerConfig := bridge.consumerConfig
	consumerConfig.Strategy = bridge.kafkaBridgeStrategy
	consumerConfig.WorkerFailureCallback = failedCallback
	consumerConfig.WorkerFailedAttemptCallback = failedAttemptCallback
	consumer := kafkaClient.NewConsumer(consumerConfig)
	topics := map[string]int{bridge.topic: consumerConfig.NumConsumerFetchers}
	go func() {
		consumer.StartStatic(topics)
	}()
	return consumer
}

func (bridge BridgeApp) kafkaBridgeStrategy(_ *kafkaClient.Worker, rawMsg *kafkaClient.Message, id kafkaClient.TaskId) kafkaClient.WorkerResult {
	msg := string(rawMsg.Value)

	go bridge.forwardMsg(msg)

	return kafkaClient.NewSuccessfulResult(id)
}

func (bridge BridgeApp) forwardMsg(kafkaMsg string) error {
	msgHeader, jsonContent, err := extractJSON(kafkaMsg)
	if err != nil {
		logger.error(fmt.Sprintf("Extracting JSON content failed. Skip forwarding message. Reason: %s", err.Error()))
		return err
	}

	logger.info(fmt.Sprintf("New message:\n---\n%s\n---", msgHeader))
	req, err := http.NewRequest("POST", "http://"+bridge.httpHost+"/notify", strings.NewReader(jsonContent))

	if err != nil {
		logger.error(fmt.Sprintf("Error creating new request: %v", err.Error()))
		return err
	}

	originSystem, err := extractOriginSystem(msgHeader)
	if err != nil {
		logger.error(fmt.Sprintf("Error parsing origin system id. Skip forwarding message. Reason: %s", err.Error()))
		return err
	}
	tid, err := extractTID(msgHeader)
	if err != nil {
		logger.warn(fmt.Sprintf("Couldn't extract transaction id: %s", err.Error()))
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		logger.info("Generating tid: " + tid)
	}

	ctxlogger := TxCombinedLogger{logger, tid}

	req.Header.Add("X-Origin-System-Id", originSystem)
	req.Header.Add("X-Request-Id", tid)
	req.Host = "cms-notifier"
	resp, err := bridge.httpClient.Do(req)
	if err != nil {
		ctxlogger.error(fmt.Sprintf("Error executing POST request to the ELB: %v", err.Error()))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Forwarding message with tid: %s is not successful. Status: %d", tid, resp.StatusCode)
		ctxlogger.error(errMsg)
		return errors.New(errMsg)
	}
	ctxlogger.info("Message forwarded")
	return nil
}

func extractJSON(msg string) (msgHeader, jsonContent string, err error) {
	startIndex := strings.Index(msg, "{")
	endIndex := strings.LastIndex(msg, "}")

	if startIndex == -1 || endIndex == -1 {
		return msgHeader, jsonContent, errors.New("Unparseable message.")
	}

	msgHeader = strings.TrimSpace(msg[:startIndex])
	jsonContent = msg[startIndex : endIndex+1]

	var temp map[string]interface{}
	err = json.Unmarshal([]byte(jsonContent), &temp)

	return msgHeader, jsonContent, err
}

var tidHeaderRegexp = regexp.MustCompile("X-Request-Id:.*")
var tidRegexp = regexp.MustCompile("(tid|SYNTHETIC-REQ-MON)[a-zA-Z0-9_-]*$")

func extractTID(msg string) (string, error) {
	header := tidHeaderRegexp.FindString(msg)
	if header == "" {
		return "", errors.New("X-Request-Id header could not be found.")
	}
	tid := tidRegexp.FindString(header)
	if tid == "" {
		return "", fmt.Errorf("Transaction ID is in unknown format: %s.", header)
	}
	return tid, nil
}

var origSysHeaderRegexp = regexp.MustCompile(`Origin-System-Id:\s[a-zA-Z0-9:/.-]*`)
var systemIDRegexp = regexp.MustCompile(`[a-zA-Z-]*$`)

func extractOriginSystem(msg string) (string, error) {
	origSysHeader := origSysHeaderRegexp.FindString(msg)
	systemID := systemIDRegexp.FindString(origSysHeader)
	if systemID == "" {
		return "", errors.New("Origin system id is not set.")
	}
	return systemID, nil
}

func failedCallback(wm *kafkaClient.WorkerManager) kafkaClient.FailedDecision {
	kafkaClient.Info("main", "Failed callback")

	return kafkaClient.DoNotCommitOffsetAndStop
}

func failedAttemptCallback(task *kafkaClient.Task, result kafkaClient.WorkerResult) kafkaClient.FailedDecision {
	kafkaClient.Info("main", "Failed attempt")

	return kafkaClient.CommitOffsetAndContinue
}

func main() {
	initLoggers()
	logger.info("Starting Kafka Bridge")
	if len(os.Args) < 2 {
		panic("Conf file path must be provided")
	}
	conf := os.Args[1]

	bridgeApp, numConsumers := newBridgeApp(conf)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	consumers := make([]*kafkaClient.Consumer, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i] = bridgeApp.startNewConsumer()
		time.Sleep(10 * time.Second)
	}

	go func() {
		http.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck", "Services: cms-notifier@aws", bridgeApp.ForwardHealthcheck()))
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			logger.error(fmt.Sprintf("Couldn't set up HTTP listener: %+v", err))
			close(ctrlc)
		}
	}()

	<-ctrlc
	logger.info("Shutdown triggered, closing all alive consumers")
	for _, consumer := range consumers {
		<-consumer.Close()
	}
	logger.info("Successfully shut down all consumers")
}
