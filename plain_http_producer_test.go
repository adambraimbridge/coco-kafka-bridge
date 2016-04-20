package main

import (
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"strings"
	"testing"
)

func TestExtractOriginSystem(t *testing.T) {
	var tests = []struct {
		msg                  queueConsumer.Message
		expectedSystemOrigin string
		expectedErrorMsg     string
	}{
		{
			queueConsumer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"methode-web-pub",
			"",
		},
		{
			queueConsumer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"",
			"Origin system id is not set",
		},
		{
			queueConsumer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"",
			"Origin system id is not set",
		},
	}

	for _, test := range tests {
		actualSystemOrigin, err := extractOriginSystem(test.msg.Headers)
		if err != nil && !strings.Contains(err.Error(), test.expectedErrorMsg) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedErrorMsg, err.Error())
		}
		if err == nil && test.expectedSystemOrigin != actualSystemOrigin {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedSystemOrigin, actualSystemOrigin)
		}
	}
}
