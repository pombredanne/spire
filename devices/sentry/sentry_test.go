package sentry_test

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/devices/sentry"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/testutils"
)

type dynamock struct {
	dynamodbiface.DynamoDBAPI
	Items []*dynamodb.PutItemInput
}

func (c *dynamock) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	if c.Items == nil {
		c.Items = []*dynamodb.PutItemInput{input}
	} else {
		c.Items = append(c.Items, input)
	}
	return nil, nil
}

var _ = Describe("Sentry Message Handler", func() {

	var broker *mqtt.Broker
	var formations *devices.FormationMap
	var recorder *testutils.PubSubRecorder
	var handler *sentry.Handler

	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"
	var ipAddress = "23.23.23.23"

	BeforeEach(func() {
		broker = mqtt.NewBroker(false)
		formations = devices.NewFormationMap()
		recorder = testutils.NewPubSubRecorder()

		handler = sentry.Register(broker, formations).(*sentry.Handler)
	})
	Describe("connect", func() {
		BeforeEach(func() {
			m := &devices.ConnectMessage{
				FormationID: formationID,
				DeviceName:  deviceName,
				DeviceInfo:  nil,
				IPAddress:   ipAddress,
			}
			broker.Publish(devices.ConnectTopic.String(), m)
		})
		It("adds the ip address to the device state", func() {
			ip := formations.GetDeviceState(deviceName, sentry.ForwardedIP)
			Expect(ip).To(Equal(ipAddress))
		})
	})
	Describe("handling messages", func() {
		var dynamo *dynamock
		var topic = "pylon/1.marsara/sentry/accept"

		BeforeEach(func() {
			formations.PutDeviceState(formationID, deviceName, sentry.ForwardedIP, ipAddress)
			dynamo = new(dynamock)
			handler.SetDynamoDBClient(dynamo)

			broker.Publish(topic, []byte(`{
				"ip": "1.2.3.4",
				"mac": "23:23:23:23:23:23",
				"timestamp": 1502982990
			}`))
		})
		It("write the data to dynamodb", func() {
			Eventually(func() int {
				return len(dynamo.Items)
			}).Should(BeNumerically("==", 1))

			item := dynamo.Items[0].Item
			data := make(map[string]interface{})
			Expect(dynamodbattribute.UnmarshalMap(item, &data)).NotTo(HaveOccurred())

			Expect(data["ip"]).To(Equal("1.2.3.4"))
			Expect(data["mac"]).To(Equal("23:23:23:23:23:23"))
			Expect(data["timestamp"]).To(BeNumerically("==", 1502982990))
			Expect(data["day"]).To(BeNumerically("==", 1503014400))
			Expect(data["pylon_ip"]).To(Equal(ipAddress))
			Expect(data["pylon_name"]).To(Equal(deviceName))
			Expect(data["action"]).To(Equal("logged_in"))
		})
	})
})
