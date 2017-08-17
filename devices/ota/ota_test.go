package ota_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/devices/ota"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/testutils"
)

var _ = Describe("OTA Message Handler", func() {

	var broker *mqtt.Broker
	var formations *devices.FormationMap
	var recorder *testutils.PubSubRecorder

	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"
	var controlTopic = "/armada/1.marsara/ota"

	BeforeEach(func() {
		broker = mqtt.NewBroker()
		formations = devices.NewFormationMap()
		recorder = testutils.NewPubSubRecorder()

		broker.Subscribe(controlTopic, recorder.Record)
		ota.Register(broker, formations)
	})
	Describe("on connect", func() {
		BeforeEach(func() {
			m := &devices.ConnectMessage{FormationID: formationID, DeviceName: deviceName, DeviceInfo: nil}
			broker.Publish(devices.ConnectTopic, m)
		})
		It("sets state to 'default'", func() {
			rawState, _ := formations.GetDeviceState(deviceName, "ota")
			state, ok := rawState.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(state.State).To(Equal(ota.Default))
		})
		It("publishes the ota message", func() {
			Expect(recorder.Count()).To(BeNumerically("==", 1))

			topic, raw := recorder.First()
			Expect(topic).To(Equal(controlTopic))

			msg, ok := raw.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(msg.State).To(Equal(ota.Default))
		})
	})
	Describe("handling messages", func() {
		BeforeEach(func() {
			formations.PutDeviceState(formationID, deviceName, "ota", &ota.Message{State: ota.Default})
		})
		JustBeforeEach(func() {
			payload := []byte(`{
					"state": "downloading",
					"progress": 10
				}`)

			broker.Publish("/pylon/1.marsara/ota", payload)
		})
		It("updates device state", func() {
			rawState, _ := formations.GetDeviceState(deviceName, "ota")
			state, ok := rawState.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(state.State).To(Equal(ota.Downloading))
		})
		It("publishes the ota message", func() {
			Expect(recorder.Count()).To(BeNumerically("==", 1))

			topic, raw := recorder.First()
			Expect(topic).To(Equal(controlTopic))

			msg, ok := raw.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(msg.State).To(Equal(ota.Downloading))
		})
	})
	Describe("on disconnect during download", func() {
		BeforeEach(func() {
			om := &ota.Message{State: ota.Downloading}
			formations.PutDeviceState(formationID, deviceName, "ota", om)

			dm := &devices.DisconnectMessage{FormationID: formationID, DeviceName: deviceName}
			broker.Publish(devices.DisconnectTopic, dm)
		})
		It("publishes an error message", func() {
			Expect(recorder.Count()).To(BeNumerically("==", 1))

			topic, raw := recorder.First()
			Expect(topic).To(Equal(controlTopic))

			msg, ok := raw.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(msg.State).To(Equal(ota.Error))
		})
	})
})
