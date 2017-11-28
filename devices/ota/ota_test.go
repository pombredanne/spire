package ota_test

import (
	"encoding/json"

	"github.com/eclipse/paho.mqtt.golang/packets"
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
	var uiRecorder *testutils.PubSubRecorder

	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"
	var topicPath = "/ota/state"
	var deviceTopic = "pylon/" + deviceName + topicPath
	var uiTopic = "matriarch/" + deviceName + topicPath

	BeforeEach(func() {
		broker = mqtt.NewBroker(false)
		formations = devices.NewFormationMap()
		uiRecorder = testutils.NewPubSubRecorder()

		broker.Subscribe(uiTopic, uiRecorder)
		ota.Register(broker, formations)
	})
	Describe("on connect", func() {
		BeforeEach(func() {
			m := devices.ConnectMessage{FormationID: formationID, DeviceName: deviceName, DeviceInfo: nil}
			broker.Publish(devices.ConnectTopic.String(), m)
		})
		It("sets state to 'default'", func() {
			formations.RLock()
			defer formations.RUnlock()

			rawState := formations.GetDeviceState(deviceName, "ota")
			state, ok := rawState.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(state.State).To(Equal(ota.Default))
		})
		It("publishes the ota state to the UI", func() {
			Expect(uiRecorder.Count()).To(BeNumerically("==", 1))

			topic, raw := uiRecorder.First()
			Expect(topic).To(Equal(uiTopic))

			msg, ok := raw.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(msg.State).To(Equal(ota.Default))
		})
	})
	Describe("handling device messages", func() {
		BeforeEach(func() {
			formations.Lock()
			defer formations.Unlock()

			formations.PutDeviceState(formationID, deviceName, "ota", &ota.Message{State: ota.Default})
		})
		JustBeforeEach(func() {
			payload := []byte(`{
					"state": "upgrading",
					"progress": 10
				}`)

			broker.Publish(deviceTopic, payload)
		})
		It("updates device state", func() {
			formations.RLock()
			defer formations.RUnlock()

			rawState := formations.GetDeviceState(deviceName, "ota")
			state, ok := rawState.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(state.State).To(Equal(ota.Upgrading))
		})
		It("publishes the ota state to the UI", func() {
			Expect(uiRecorder.Count()).To(BeNumerically("==", 1))

			topic, raw := uiRecorder.First()
			Expect(topic).To(Equal(uiTopic))

			msg, ok := raw.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(msg.State).To(Equal(ota.Upgrading))
		})
	})
	Describe("handling control messages", func() {
		var controlTopic, deviceTopic string
		var deviceRecorder *testutils.PubSubRecorder
		var payload []byte

		JustBeforeEach(func() {
			deviceRecorder = testutils.NewPubSubRecorder()
			broker.Subscribe(deviceTopic, deviceRecorder)
			broker.Publish(controlTopic, payload)
		})
		Context("'sysupgrade'", func() {
			BeforeEach(func() {
				controlTopic = "armada/" + deviceName + "/ota/sysupgrade"
				deviceTopic = "pylon/" + deviceName + "/ota/sysupgrade"

				payload = []byte(`{
					"url": "http://your.shiny/new/image",
					"sha256": "ab63bd5c3377e8d4fd4e16ae3ef24236b4008d4a2ae10a516aabd17a62df97fc"
				}`)
			})
			It("updates device state", func() {
				formations.RLock()
				defer formations.RUnlock()

				rawState := formations.GetDeviceState(deviceName, "ota")
				state, ok := rawState.(*ota.Message)
				Expect(ok).To(BeTrue())
				Expect(state.State).To(Equal(ota.Downloading))
			})
			It("publishes the ota state to the UI", func() {
				Expect(uiRecorder.Count()).To(BeNumerically("==", 1))

				topic, raw := uiRecorder.First()
				Expect(topic).To(Equal(uiTopic))

				msg, ok := raw.(*ota.Message)
				Expect(ok).To(BeTrue())
				Expect(msg.State).To(Equal(ota.Downloading))
			})
			It("forwards the message to the device", func() {
				Expect(deviceRecorder.Count()).To(BeNumerically("==", 1))

				topic, raw := deviceRecorder.First()
				Expect(topic).To(Equal(deviceTopic))

				buf, ok := raw.([]byte)
				Expect(ok).To(BeTrue())

				m := map[string]string{}
				err := json.Unmarshal(buf, &m)
				Expect(err).NotTo(HaveOccurred())

				Expect(m["url"]).To(Equal("http://your.shiny/new/image"))
				Expect(m["sha256"]).To(Equal("ab63bd5c3377e8d4fd4e16ae3ef24236b4008d4a2ae10a516aabd17a62df97fc"))
			})
		})
		Context("'cancel'", func() {
			BeforeEach(func() {
				controlTopic = "armada/" + deviceName + "/ota/cancel"
				deviceTopic = "pylon/" + deviceName + "/ota/cancel"
				payload = []byte("whatevs")
			})
			It("updates device state", func() {
				formations.RLock()
				defer formations.RUnlock()

				rawState := formations.GetDeviceState(deviceName, "ota")
				state, ok := rawState.(*ota.Message)
				Expect(ok).To(BeTrue())
				Expect(state.State).To(Equal(ota.Cancelled))
			})
			It("publishes the ota state to the UI", func() {
				Expect(uiRecorder.Count()).To(BeNumerically("==", 1))

				topic, raw := uiRecorder.First()
				Expect(topic).To(Equal(uiTopic))

				msg, ok := raw.(*ota.Message)
				Expect(ok).To(BeTrue())
				Expect(msg.State).To(Equal(ota.Cancelled))
			})
			It("forwards the message to the device", func() {
				Expect(deviceRecorder.Count()).To(BeNumerically("==", 1))

				topic, _ := deviceRecorder.First()
				Expect(topic).To(Equal(deviceTopic))
			})
		})
	})
	Describe("on disconnect during download", func() {
		BeforeEach(func() {
			formations.Lock()
			om := &ota.Message{State: ota.Downloading}
			formations.PutDeviceState(formationID, deviceName, "ota", om)
			formations.Unlock()

			dm := devices.DisconnectMessage{FormationID: formationID, DeviceName: deviceName}
			broker.Publish(devices.DisconnectTopic.String(), dm)
		})
		It("publishes an error message to the UI", func() {
			Expect(uiRecorder.Count()).To(BeNumerically("==", 1))

			topic, raw := uiRecorder.First()
			Expect(topic).To(Equal(uiTopic))

			msg, ok := raw.(*ota.Message)
			Expect(ok).To(BeTrue())
			Expect(msg.State).To(Equal(ota.Error))
		})
	})
	Describe("sends current OTA state on subscribe", func() {
		var brokerSession, subscriberSession *mqtt.Session
		var payload map[string]interface{}

		JustBeforeEach(func() {
			brokerSession, subscriberSession = testutils.Pipe()

			subPkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
			subPkg.Topics = []string{"matriarch/1.marsara/#"}
			subPkg.MessageID = 1337

			go broker.HandleSubscribePacket(subPkg, brokerSession, true)

			// read suback packet
			_, err := subscriberSession.Read()
			Expect(err).NotTo(HaveOccurred())

			p, err := subscriberSession.Read()
			Expect(err).NotTo(HaveOccurred())

			pubPkg, ok := p.(*packets.PublishPacket)
			Expect(ok).To(BeTrue())
			payload = make(map[string]interface{})
			err = json.Unmarshal(pubPkg.Payload, &payload)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			brokerSession.Close()
			subscriberSession.Close()
		})
		Context("with no device state in the cache", func() {
			It("publishes an OTA message for the device with \"default\" state", func() {
				Expect(payload["state"]).To(Equal("default"))
			})
		})
		Context("with device state in the cache", func() {
			BeforeEach(func() {
				formations.Lock()
				defer formations.Unlock()

				stateMsg := &ota.Message{State: ota.Downloading, Progress: 42}
				formations.PutDeviceState(formationID, deviceName, "ota", stateMsg)
			})
			It("publishes an OTA message for the device with \"downloading\" state", func() {
				Expect(payload["state"]).To(Equal("downloading"))
			})
		})
	})
})
