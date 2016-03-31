package main

import (
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"regexp"
	"strings"
	"testing"
)

func TestExtractTID(t *testing.T) {
	var tests = []struct {
		msg                   queueConsumer.Message
		expectedTransactionID string
		expectedErrorMsg      string
	}{
		{queueConsumer.Message{
			Headers: map[string]string{
				"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
				"Message-Timestamp": "2015-07-06T07:03:09.362Z",
				"Message-Type":      "cms-content-published",
				"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
				"Content-Type":      "application/json",
				"X-Request-Id":      "tid_t9happe59y",
			},
			Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"tid_t9happe59y",
			"",
		},
		{
			queueConsumer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"",
			"X-Request-Id header could not be found",
		},
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
			"",
			"Transaction ID is in unknown format",
		},
	}

	for _, test := range tests {
		actualTransactionID, err := extractTID(test.msg.Headers)
		if err != nil && !strings.Contains(err.Error(), test.expectedErrorMsg) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedErrorMsg, err.Error())
		}
		if err == nil && test.expectedTransactionID != actualTransactionID {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedTransactionID, actualTransactionID)
		}
	}
}

func TestExtractTID_TIDRegexp(t *testing.T) {
	var tests = []struct {
		header string
		tid    string
	}{
		{"X-Request-Id:tid_ABCDe12345", "tid_ABCDe12345"},
		{"X-Request-Id: tid_ABCDe12345", "tid_ABCDe12345"},
		{"X-Request-Id: SYNTHETIC-REQ-MON_ABCDe12345", "SYNTHETIC-REQ-MON_ABCDe12345"},
		{"X-Request-Id: SYNTHETIC-REQ-MON_ABCDe12345", "SYNTHETIC-REQ-MON_ABCDe12345"},
		{"X-Request-Id: SYNTHETIC-REQ-MON-abcdefgh-1234-pqrs-5678-stuvwxyz", "SYNTHETIC-REQ-MON-abcdefgh-1234-pqrs-5678-stuvwxyz"},
		{"X-Request-Id: SYNTHETIC-REQ-MONabcdefgh-1234-pqrs-5678-stuvwxyz", "SYNTHETIC-REQ-MONabcdefgh-1234-pqrs-5678-stuvwxyz"},
		{"X-Request-Id: SYN-REQ-MON_ABCDe12345", ""},
		{"X-Request-Id: ABCDE12345", ""},
		{"X-Request-Id: tid_ABCDe1234%", ""},
	}
	validRegexp := regexp.MustCompile(tidValidRegexp)
	for _, test := range tests {

		actualTID := validRegexp.FindString(test.header)
		if actualTID != test.tid {
			t.Errorf("\nHeader: %s\nExpectedTID: %s\nActualTID: %s\n", test.header, test.tid, actualTID)
		}
	}
}

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

func TestExtractUUID(t *testing.T) {
	var tests = []struct {
		msg              queueConsumer.Message
		expectedUUID     string
		expectedErrorMsg string
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
			"7543220a-2389-11e5-bd83-71cb60e8f08c",
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
				Body: `{"type":"EOM::CompoundStory","value":"test"}`},
			"",
			"UUID is not present",
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
			"7543220a-2389-11e5-bd83-71cb60e8f08c",
			"",
		},
	}

	for _, test := range tests {
		actualUUID, err := extractUUID(test.msg.Body)
		if err != nil && !strings.Contains(err.Error(), test.expectedErrorMsg) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedErrorMsg, err.Error())
		}
		if err == nil && (test.expectedErrorMsg != "" || test.expectedUUID != actualUUID) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedUUID, actualUUID)
		}
	}
}