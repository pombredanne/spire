package devices_test

import (
	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/devices/deviceInfo"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/testutils"
)

var _ = Describe("Device Message Handlers", func() {

	var broker *mqtt.Broker
	var formations *devices.FormationMap

	var devMsgHandler *devices.Handler
	var deviceServer, deviceClient *mqtt.Session
	var response packets.ControlPacket

	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"

	BeforeEach(func() {
		broker = mqtt.NewBroker()
		formations = devices.NewFormationMap()
		devMsgHandler = devices.NewHandler(broker)
		deviceServer, deviceClient = testutils.Pipe()
	})
	JustBeforeEach(func() {
		go func() {
			devMsgHandler.HandleConnection(deviceServer)
		}()

		Expect(testutils.WriteConnectPacket(formationID, deviceName, "", deviceClient)).NotTo(HaveOccurred())

		var err error
		response, err = deviceClient.Read()
		Expect(err).NotTo(HaveOccurred())
	})
	Describe("connect", func() {
		It("sends CONNACK", func() {
			_, ok := response.(*packets.ConnackPacket)
			Expect(ok).To(BeTrue())
		})
		Describe("device info", func() {
			BeforeEach(func() {
				deviceInfo.Register(broker, formations)
			})
			It("fetches device info and adds 'device_os' to device state", func() {
				var deviceInfoState interface{}

				Eventually(func() interface{} {
					deviceInfoState, _ = formations.GetDeviceState(deviceName, "device_info")
					return deviceInfoState
				}).ShouldNot(BeNil())

				Expect(deviceInfoState.(map[string]interface{})["device_os"]).To(Equal("tplink-archer-c7-lingrush-44"))
			})
		})
		Describe("pub/sub", func() {
			var recorder *testutils.PubSubRecorder

			BeforeEach(func() {
				recorder = testutils.NewPubSubRecorder()
				broker.Subscribe(devices.ConnectTopic, recorder.Record)
			})
			It("publishes a connect message for the device", func() {
				Eventually(func() int {
					return recorder.Count()
				}).Should(BeNumerically("==", 1))

				topic, raw := recorder.First()
				Expect(topic).To(Equal(devices.ConnectTopic))

				cm, ok := raw.(*devices.ConnectMessage)
				Expect(ok).To(BeTrue())

				Expect(cm.FormationID).To(Equal(formationID))
				Expect(cm.DeviceName).To(Equal(deviceName))
				Expect(cm.DeviceInfo).ToNot(BeNil())
				Expect(cm.DeviceInfo["data"]).ToNot(BeNil())
			})
		})
	})
	Describe("disconnect", func() {
		var recorder *testutils.PubSubRecorder

		BeforeEach(func() {
			recorder = testutils.NewPubSubRecorder()
			broker.Subscribe(devices.DisconnectTopic, recorder.Record)
		})
		Context("by sending DISCONNECT", func() {
			JustBeforeEach(func() {
				p := packets.NewControlPacket(packets.Disconnect)
				go deviceClient.Write(p)
			})
			It("publishes a disconnect message for the device", func() {
				Eventually(func() int {
					return recorder.Count()
				}).Should(BeNumerically("==", 1))

				topic, raw := recorder.First()
				Expect(topic).To(Equal(devices.DisconnectTopic))

				cm, ok := raw.(*devices.DisconnectMessage)
				Expect(ok).To(BeTrue())

				Expect(cm.FormationID).To(Equal(formationID))
				Expect(cm.DeviceName).To(Equal(deviceName))
			})
		})
		Context("by closing the connection", func() {
			JustBeforeEach(func() {
				Expect(deviceClient.Close()).ToNot(HaveOccurred())
			})
			It("publishes a disconnect message for the device", func() {
				Eventually(func() int {
					return recorder.Count()
				}).Should(BeNumerically("==", 1))

				topic, raw := recorder.First()
				Expect(topic).To(Equal(devices.DisconnectTopic))

				cm, ok := raw.(*devices.DisconnectMessage)
				Expect(ok).To(BeTrue())

				Expect(cm.FormationID).To(Equal(formationID))
				Expect(cm.DeviceName).To(Equal(deviceName))
			})
		})
	})
})
