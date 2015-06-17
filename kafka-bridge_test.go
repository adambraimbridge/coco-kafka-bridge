package main

import (
    "fmt"
    "io/ioutil"
    "testing"
)

func TestParse(t *testing.T) {

    buf, err := ioutil.ReadFile("test_msg.txt")
    if (err != nil) {
        fmt.Print("Couldn't read content from file.")
    }
    msg := string(buf)

    buf, err = ioutil.ReadFile("test_content.txt")
    if (err != nil) {
        fmt.Print("Couldn't read content from file.")
    }
    expectedContent := string(buf)

    actualContent := parseMsg(msg);
    if (expectedContent != actualContent) {
        t.Errorf("not equal")
    }
}
