package sentry

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/superscale/spire/config"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

// ForwardedIP is the key used for storing/retrieving the public IP address of a device
const ForwardedIP = "forwarded_ip"

// Message ...
type Message struct {
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	Timestamp time.Time `json:"timestamp"`
}

// Handler ...
type Handler struct {
	formations     *devices.FormationMap
	awsSession     *session.Session
	dynamoDBClient dynamodbiface.DynamoDBAPI
}

// Register ...
func Register(broker *mqtt.Broker, formations *devices.FormationMap) interface{} {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(endpoints.EuWest1RegionID),
	}))

	h := &Handler{
		formations:     formations,
		awsSession:     sess,
		dynamoDBClient: dynamodb.New(sess),
	}

	broker.Subscribe(devices.ConnectTopic, h.onConnect)
	broker.Subscribe("/pylon/#/sentry/accept", h.onMessage)
	return h
}

func (h *Handler) onConnect(topic string, payload interface{}) error {
	cm := payload.(*devices.ConnectMessage)
	h.formations.PutDeviceState(cm.FormationID, cm.DeviceName, ForwardedIP, cm.IPAddress)
	return nil
}

func (h *Handler) onMessage(topic string, payload interface{}) error {
	t := devices.ParseTopic(topic)

	buf, ok := payload.([]byte)
	if !ok {
		return fmt.Errorf("[sentry] expected []byte, got this instead: %v", payload)
	}

	var m Message
	if err := json.Unmarshal(buf, &m); err != nil {
		return err
	}

	ts := m.Timestamp.Unix()

	item, err := dynamodbattribute.MarshalMap(map[string]interface{}{
		"ip":         m.IP,
		"mac":        m.MAC,
		"timestamp":  ts,
		"day":        devices.Round(float64(ts)/86400.0, 0) * 86400,
		"pylon_ip":   h.getForwardedIP(t.DeviceName),
		"pylon_name": t.DeviceName,
		"action":     "logged_in",
	})
	if err != nil {
		return err
	}

	_, err = h.dynamoDBClient.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(config.Config.SentryDynamoDBTable),
		Item:      item,
	})
	return err
}

func (h *Handler) getForwardedIP(deviceName string) string {
	raw := h.formations.GetDeviceState(deviceName, ForwardedIP)
	if raw != nil {
		if ip, ok := raw.(string); ok {
			return ip
		}
	}
	return "unknown"
}

// SetDynamoDBClient is used in tests to mock DynamoDB operations
func (h *Handler) SetDynamoDBClient(client dynamodbiface.DynamoDBAPI) {
	h.awsSession = nil
	h.dynamoDBClient = client
}

// UnmarshalJSON ...
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	tmp := &struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	m.Timestamp = time.Unix(tmp.Timestamp, 0)
	return nil
}
