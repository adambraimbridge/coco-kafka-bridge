package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	ftHealth "github.com/Financial-Times/go-fthealth"
)

var httpClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 100,
		Dial: (&net.Dialer{
			KeepAlive: 30 * time.Second,
		}).Dial,
	}}

func (bridge BridgeApp) consumeHealthcheck() ftHealth.Check {
	return ftHealth.Check{
		BusinessImpact:   "Consuming messages through kafka-proxy won't work. Publishing in the containerised stack won't work.",
		Name:             "Consume from UCS kafka through the proxy",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Consuming messages is broken. Check if kafka-proxy in aws is reachable.",
		Checker:          bridge.aggregateConsumableResults,
	}
}

func (bridge BridgeApp) proxyForwarderHealthcheck() ftHealth.Check {
	return ftHealth.Check{
		BusinessImpact:   "Forwarding messages to kafka-proxy in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to kafka-proxy.",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check if kafka-proxy in coco is reachable.",
		Checker:          bridge.checkForwardableProxy,
	}
}

func (bridge BridgeApp) httpForwarderHealthcheck() ftHealth.Check {
	return ftHealth.Check{
		BusinessImpact:   "Forwarding messages to cms-notifier in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to cms-notifier",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check networking, aws cluster reachability and/or coco cms-notifier state.",
		Checker:          bridge.checkForwardableHTTP,
	}
}

func (bridge BridgeApp) aggregateConsumableResults() error {
	addresses := bridge.consumerConfig.Addrs
	errMsg := ""
	for i := 0; i < len(addresses); i++ {
		err := bridge.checkConsumable(addresses[i])
		if err == nil {
			return nil
		}
		errMsg = errMsg + fmt.Sprintf("For %s there is an error %v \n", addresses[i], err.Error())
	}

	return errors.New(errMsg)
}

func (bridge BridgeApp) checkConsumable(address string) error {
	body, err := checkProxyConnection(address, bridge.consumerConfig.AuthorizationKey)
	if err != nil {
		logger.error(fmt.Sprintf("Healthcheck: Error reading request body: %v", err.Error()))
		return err
	}
	return checkIfTopicIsPresent(body, bridge.consumerConfig.Topic)
}

func checkProxyConnection(address string, authorizationKey string) ([]byte, error) {
	//check if proxy is running and topic is present
	req, err := http.NewRequest("GET", address+"/topics", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new kafka-proxy healthcheck request: %v", err.Error()))
		return nil, err
	}

	if authorizationKey != "" {
		req.Header.Add("Authorization", authorizationKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.error(fmt.Sprintf("Healthcheck: Error executing kafka-proxy GET request: %v", err.Error()))
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Connecting to kafka proxy was not successful. Status: %d", resp.StatusCode)
		return nil, errors.New(errMsg)
	}

	return ioutil.ReadAll(resp.Body)
}

func checkIfTopicIsPresent(body []byte, searchedTopic string) error {
	var topics []string

	err := json.Unmarshal(body, &topics)
	if err != nil {
		return fmt.Errorf("Connection could be established to kafka-proxy, but a parsing error occured and topic could not be found. %v", err.Error())
	}

	for _, topic := range topics {
		if topic == searchedTopic {
			return nil
		}
	}

	return errors.New("Connection could be established to kafka-proxy, but topic was not found")
}

func (bridge BridgeApp) checkForwardableHTTP() error {
	req, err := http.NewRequest("GET", bridge.producerConfig.Addr+"/__health", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new plainHttp producer healthcheck request: %v", err.Error()))
		return err
	}
	req.Header.Add("Authorization", bridge.producerConfig.Authorization)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.warn(fmt.Sprintf("Healthcheck: Error executing GET request: %v", err.Error()))
		return err
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Healthcheck: Request to plainHTTP producer /__health endpoint failed. Status: %d.", resp.StatusCode)
		logger.warn(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (bridge BridgeApp) checkForwardableProxy() error {
	_, err := checkProxyConnection(bridge.producerConfig.Addr, bridge.producerConfig.Authorization)
	return err
}
