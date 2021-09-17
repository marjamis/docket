package cmd

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestFormatDDBEntry(t *testing.T) {
	// TODO fix up this test
	dat, err := ioutil.ReadFile("../test/container_instance_state_change_event.json")
	if err != nil {
		panic(err)
	}

	var event events.CloudWatchEvent
	err = json.Unmarshal([]byte(dat), &event)
	if err != nil {
		panic(err)
	}

	resp := formatDDBEntry(event)
	te := true
	r := &dynamodb.AttributeValue{BOOL: &te}
	assert.Equal(t, r, resp["agentConnected"])
	se := "ACTIVE"
	s := &dynamodb.AttributeValue{S: &se}
	assert.Equal(t, s, resp["status"])
}

func TestDDBPutItem(t *testing.T) {
	// TODO test passing in the client and test that whole thing from a more dependency injection perspective - though need outside of handler
}
