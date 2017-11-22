package up_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/devices/up"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/testutils"
)

var _ = Describe("Up Message Handler", func() {

	var broker *mqtt.Broker
	var formations *devices.FormationMap
	var recorder *testutils.PubSubRecorder

	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"
	var upTopic = "matriarch/1.marsara/up"

	BeforeEach(func() {
		broker = mqtt.NewBroker(false)
		formations = devices.NewFormationMap()
		recorder = testutils.NewPubSubRecorder()

		broker.Subscribe(upTopic, recorder)
		up.Register(broker, formations)
	})
	Describe("connect", func() {
		BeforeEach(func() {
			m := devices.ConnectMessage{FormationID: formationID, DeviceName: deviceName, DeviceInfo: nil}
			broker.Publish(devices.ConnectTopic.String(), m)
		})
		It("publishes an 'up' message for the device with state = \"up\"", func() {
			Eventually(func() int {
				return recorder.Count()
			}).Should(BeNumerically("==", 1))

			topic, raw := recorder.First()

			Expect(topic).To(Equal(upTopic))

			msg, ok := raw.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(msg["state"]).To(Equal("up"))

			timestamp, ok := msg["timestamp"]
			Expect(ok).To(BeTrue())
			_, ok = timestamp.(int64)
			Expect(ok).To(BeTrue())
		})
		Describe("disconnect", func() {
			BeforeEach(func() {
				m := devices.DisconnectMessage{FormationID: formationID, DeviceName: deviceName}
				broker.Publish(devices.DisconnectTopic.String(), m)
			})
			It("publishes an 'up' message for the device with state = \"down\"", func() {
				Eventually(func() int {
					return recorder.Count()
				}).Should(BeNumerically("==", 2))

				topic, raw := recorder.Last()
				Expect(topic).To(Equal(upTopic))

				msg, ok := raw.(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(msg["state"]).To(Equal("down"))

				timestamp, ok := msg["timestamp"]
				Expect(ok).To(BeTrue())
				_, ok = timestamp.(int64)
				Expect(ok).To(BeTrue())
			})
		})
	})
})
