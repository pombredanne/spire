package devices_test

import (
	"fmt"

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
		broker = mqtt.NewBroker(false)
		formations = devices.NewFormationMap()
		devMsgHandler = devices.NewHandler(formations, broker)
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
		It("stores formation ID for device", func() {
			Expect(formations.FormationID(deviceName)).To(Equal(formationID))
		})
		Describe("device info", func() {
			BeforeEach(func() {
				deviceInfo.Register(broker, formations)
			})
			It("fetches device info and adds 'device_os' to device state", func() {
				var deviceInfoState interface{}

				Eventually(func() interface{} {
					deviceInfoState = formations.GetDeviceState(deviceName, "device_info")
					return deviceInfoState
				}).ShouldNot(BeNil())

				Expect(deviceInfoState.(map[string]interface{})["device_os"]).To(Equal("tplink-archer-c7-lingrush-44"))
			})
		})
		Describe("pub/sub", func() {
			var recorder *testutils.PubSubRecorder

			BeforeEach(func() {
				recorder = testutils.NewPubSubRecorder()
				broker.Subscribe(devices.ConnectTopic.String(), recorder)
			})
			It("publishes a connect message for the device", func() {
				Eventually(func() int {
					return recorder.Count()
				}).Should(BeNumerically("==", 1))

				topic, raw := recorder.First()
				Expect(topic).To(Equal(devices.ConnectTopic.String()))

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
			broker.Subscribe(devices.DisconnectTopic.String(), recorder)
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
				Expect(topic).To(Equal(devices.DisconnectTopic.String()))

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
				Expect(topic).To(Equal(devices.DisconnectTopic.String()))

				cm, ok := raw.(*devices.DisconnectMessage)
				Expect(ok).To(BeTrue())

				Expect(cm.FormationID).To(Equal(formationID))
				Expect(cm.DeviceName).To(Equal(deviceName))
			})
		})
	})
	Describe("ParseTopic", func() {
		var prefix string
		var path string
		var result devices.Topic

		Context("regular topic", func() {
			BeforeEach(func() {
				path = "wifi/poll"
			})
			JustBeforeEach(func() {
				result = devices.ParseTopic(fmt.Sprintf("%s/%s/%s", prefix, deviceName, path))
			})
			Context("with leading slash", func() {
				BeforeEach(func() {
					prefix = "/pylon"
				})
				It("separates the topic into prefix, device name and path", func() {
					Expect(result.Prefix).To(Equal(prefix[1:]))
					Expect(result.DeviceName).To(Equal(deviceName))
					Expect(result.Path).To(Equal(path))
				})
			})
			Context("without leading slash", func() {
				BeforeEach(func() {
					prefix = "pylon"
				})
				It("separates the topic into prefix, device name and path", func() {
					Expect(result.Prefix).To(Equal(prefix))
					Expect(result.DeviceName).To(Equal(deviceName))
					Expect(result.Path).To(Equal(path))
				})
			})
		})
		Context("internal topic", func() {
			JustBeforeEach(func() {
				path = "spire/devices/connect"
				result = devices.ParseTopic(fmt.Sprintf("%s/%s", prefix, path))
			})
			Context("with leading slash", func() {
				BeforeEach(func() {
					prefix = "/" + mqtt.InternalTopicPrefix
				})
				It("separates the topic into prefix and path", func() {
					Expect(result.Prefix).To(Equal(prefix[1:]))
					Expect(result.DeviceName).To(BeEmpty())
					Expect(result.Path).To(Equal(path))
				})
			})
			Context("without leading slash", func() {
				BeforeEach(func() {
					prefix = mqtt.InternalTopicPrefix
				})
				It("separates the topic into prefix and path", func() {
					Expect(result.Prefix).To(Equal(prefix))
					Expect(result.DeviceName).To(BeEmpty())
					Expect(result.Path).To(Equal(path))
				})
			})
		})
	})
})
