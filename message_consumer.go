package main

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
)

func (bridge BridgeApp) consumeMessages() {
	consumerConfig := bridge.consumerConfig

	consumer := queueConsumer.NewAgeingConsumer(*consumerConfig, bridge.forwardMsg, queueConsumer.AgeingClient{
		Client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 100,
				Dial: (&net.Dialer{
					KeepAlive: 30 * time.Second,
				}).Dial,
			},
		},
		MaxAge: time.Duration(2) * time.Minute,
	})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		consumer.Start()
		wg.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	consumer.Stop()
	wg.Wait()
}
