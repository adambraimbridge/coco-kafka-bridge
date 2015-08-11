package main

import (
	"strings"
	"testing"
)

func TestExtractJSON(t *testing.T) {

	var tests = []struct {
		kafkaMsg            string
		expectedMsgHeader   string
		expectedJSONContent string
	}{
		{
			`
            FTMSG/1.0
            Message-Id: bb07b9ab-0ff6-4853-bdd1-104906d7d282
            Message-Timestamp: 2015-06-17T12:16:39.022Z
            Message-Type: cms-content-published
            Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
            Content-Type: application/json
            X-Request-Id: tid_6y3oogjqhk

            { "uuid":"f9d6eecc-14b4-11e5-973e-a0f360779259","type":"EOM::CompoundStory","value":"bodor_kafka_bridge_test","attributes":[],"linkedObjects":[] }
            `,
			`FTMSG/1.0
            Message-Id: bb07b9ab-0ff6-4853-bdd1-104906d7d282
            Message-Timestamp: 2015-06-17T12:16:39.022Z
            Message-Type: cms-content-published
            Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
            Content-Type: application/json
            X-Request-Id: tid_6y3oogjqhk`,
			`{ "uuid":"f9d6eecc-14b4-11e5-973e-a0f360779259","type":"EOM::CompoundStory","value":"bodor_kafka_bridge_test","attributes":[],"linkedObjects":[] }`,
		},
	}

	for _, test := range tests {
		actualMsgHeader, actualJSONContent, err := extractJSON(test.kafkaMsg)
		if err != nil || test.expectedJSONContent != actualJSONContent || test.expectedMsgHeader != actualMsgHeader {
			t.Errorf("\nExpected msg header: %s\nActual msg header: %s\nExpected JSON: %s\nActual JSON: %s", test.expectedMsgHeader, actualMsgHeader, test.expectedJSONContent, actualJSONContent)
		}
	}
}

func TestExtractTID(t *testing.T) {
	var tests = []struct {
		msg                   string
		expectedTransactionID string
		expectedErrorMsg      string
	}{
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
			Content-Type: application/json
			X-Request-Id: tid_t9happe59y

			{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}
			`,
			"tid_t9happe59y",
			"",
		},
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
			Content-Type: application/json

			{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}
			`,
			"",
			"X-Request-Id header could not be found",
		},
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
			Content-Type: application/json
			X-Request-Id: t9happe59y

			{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}
			`,
			"",
			"Transaction ID is in unknown format",
		},
	}

	for _, test := range tests {
		actualTransactionID, err := extractTID(test.msg)
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
		{"X-Request-Id: SYN-REQ-MON_ABCDe12345", "SYN-REQ-MON_ABCDe12345"},
		{"X-Request-Id:  SYN-REQ-MON_ABCDe12345", "SYN-REQ-MON_ABCDe12345"},
		{"X-Request-Id: ABCDE12345", ""},
		{"X-Request-Id: tid_ABCDe1234%", ""},
	}

	for _, test := range tests {
		actualTID := tidRegexp.FindString(test.header)
		if actualTID != test.tid {
			t.Errorf("\nHeader: %s\nExpectedTID: %s\nActualTID: %s\n", test.header, test.tid, actualTID)
		}
	}
}

func TestExtractOriginSystem(t *testing.T) {
	var tests = []struct {
		msg                  string
		expectedSystemOrigin string
		expectedErrorMsg     string
	}{
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
			Content-Type: application/json
			X-Request-Id: t9happe59y
			`,
			"methode-web-pub",
			"",
		},
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Content-Type: application/json
			X-Request-Id: t9happe59y
			`,
			"",
			"Origin system id is not set",
		},
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id:
			Content-Type: application/json
			X-Request-Id: t9happe59y
			`,
			"",
			"Origin system id is not set",
		},
	}

	for _, test := range tests {
		actualSystemOrigin, err := extractOriginSystem(test.msg)
		if err != nil && !strings.Contains(err.Error(), test.expectedErrorMsg) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedErrorMsg, err.Error())
		}
		if err == nil && test.expectedSystemOrigin != actualSystemOrigin {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedSystemOrigin, actualSystemOrigin)
		}
	}
}
