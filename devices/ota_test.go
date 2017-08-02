package devices_test

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

var _ = Describe("OTA Message Handler", func() {
	var broker *mqtt.Broker
	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"
	var topic = fmt.Sprintf("/pylon/%s/ota", deviceName)
	var payload []byte

	BeforeEach(func() {
		broker = mqtt.NewBroker()
	})
	Describe("on connect", func() {
		var devMsgHandler *devices.MessageHandler
		var deviceServer, deviceClient net.Conn
		var connected chan bool

		JustBeforeEach(func() {
			devMsgHandler = devices.NewMessageHandler(broker)
			deviceServer, deviceClient = net.Pipe()

			Expect(devMsgHandler.GetDeviceState(formationID, deviceName, "ota")).To(BeNil())

			go func() {
				devMsgHandler.HandleConnection(deviceServer)
			}()

			connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
			connPkg.ClientIdentifier = deviceName
			connPkg.Username = formationID
			connPkg.UsernameFlag = true
			Expect(connPkg.Write(deviceClient)).NotTo(HaveOccurred())

			connected = make(chan bool)
			go func() {
				_, err := packets.ReadPacket(deviceClient)
				Expect(err).NotTo(HaveOccurred())
				connected <- true
			}()
		})
		It("sets state to 'default'", func() {
			<-connected
			otaState, ok := devMsgHandler.GetDeviceState(formationID, deviceName, "ota").(*devices.OTAState)
			Expect(ok).To(BeTrue())
			Expect(otaState.State).To(Equal(devices.Default))
			deviceClient.Close()
		})
		Context("with subscribers", func() {
			var subscriberConn, brokerConn net.Conn

			BeforeEach(func() {
				subscriberConn, brokerConn = net.Pipe()
				subPkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
				subPkg.Topics = []string{"/armada/" + deviceName + "/ota"}
				subPkg.Qoss = []byte{0}
				go broker.Subscribe(subPkg, brokerConn)

				pkg, err := packets.ReadPacket(subscriberConn)
				Expect(err).NotTo(HaveOccurred())
				_, ok := pkg.(*packets.SubackPacket)
				Expect(ok).To(BeTrue())
			})
			It("publishes the ota message", func() {
				var ok bool
				var pubPkg *packets.PublishPacket
				pubRead := make(chan bool)

				go func() {
					pkg, err := packets.ReadPacket(subscriberConn)
					Expect(err).NotTo(HaveOccurred())
					pubPkg, ok = pkg.(*packets.PublishPacket)
					Expect(ok).To(BeTrue())
					pubRead <- true
				}()

				<-pubRead
				Expect(pubPkg.TopicName).To(Equal("/armada/" + deviceName + "/ota"))

				var otaState devices.OTAState
				err := json.Unmarshal(pubPkg.Payload, &otaState)
				Expect(err).NotTo(HaveOccurred())
				Expect(otaState.State).To(Equal(devices.Default))
			})
		})
	})
	Describe("handling messages", func() {
		var formations *devices.FormationMap
		var done chan bool

		BeforeEach(func() {
			done = make(chan bool)
			formations = devices.NewFormationMap()
			formations.PutDeviceState(formationID, deviceName, "ota", &devices.OTAState{State: devices.Default})

			payload = []byte(`{
					"state": "downloading",
					"progress": 10
				}`)
		})
		JustBeforeEach(func() {
			go func() {
				err := devices.HandleOTA(topic, payload, formationID, deviceName, formations, broker)
				Expect(err).ToNot(HaveOccurred())
				done <- true
			}()
		})
		It("updates device state", func() {
			<-done
			otaState, ok := formations.GetDeviceState(formationID, deviceName, "ota").(*devices.OTAState)
			Expect(ok).To(BeTrue())
			Expect(otaState.State).To(Equal(devices.Downloading))
		})
		Context("ota with subscribers", func() {
			var subscriberConn, brokerConn net.Conn

			BeforeEach(func() {
				subscriberConn, brokerConn = net.Pipe()
				subPkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
				subPkg.Topics = []string{"/armada/" + deviceName + "/ota"}
				subPkg.Qoss = []byte{0}
				go broker.Subscribe(subPkg, brokerConn)

				pkg, err := packets.ReadPacket(subscriberConn)
				Expect(err).NotTo(HaveOccurred())
				_, ok := pkg.(*packets.SubackPacket)
				Expect(ok).To(BeTrue())
			})
			It("publishes the ota message", func() {
				var ok bool
				var pubPkg *packets.PublishPacket
				pubRead := make(chan bool)

				go func() {
					pkg, err := packets.ReadPacket(subscriberConn)
					Expect(err).NotTo(HaveOccurred())
					pubPkg, ok = pkg.(*packets.PublishPacket)
					Expect(ok).To(BeTrue())
					pubRead <- true
				}()

				<-pubRead
				Expect(pubPkg.TopicName).To(Equal("/armada/" + deviceName + "/ota"))

				var otaState devices.OTAState
				err := json.Unmarshal(pubPkg.Payload, &otaState)
				Expect(err).NotTo(HaveOccurred())
				Expect(otaState.State).To(Equal(devices.Downloading))
			})
		})
	})
	Describe("on disconnect during download", func() {
		var devMsgHandler *devices.MessageHandler
		var deviceServer, deviceClient, subscriberConn, brokerConn net.Conn
		var connected chan bool

		JustBeforeEach(func() {
			devMsgHandler = devices.NewMessageHandler(broker)
			deviceServer, deviceClient = net.Pipe()

			Expect(devMsgHandler.GetDeviceState(formationID, deviceName, "ota")).To(BeNil())

			go devMsgHandler.HandleConnection(deviceServer)

			connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
			connPkg.ClientIdentifier = deviceName
			connPkg.Username = formationID
			connPkg.UsernameFlag = true
			Expect(connPkg.Write(deviceClient)).NotTo(HaveOccurred())

			connected = make(chan bool)
			go func() {
				_, err := packets.ReadPacket(deviceClient)
				Expect(err).NotTo(HaveOccurred())

				pkg, err := mqtt.MakePublishPacket(topic, []byte(`{"state": "downloading"}`))
				Expect(err).NotTo(HaveOccurred())
				pkg.Write(deviceClient)
				Expect(err).NotTo(HaveOccurred())

				connected <- true
			}()

			subscriberConn, brokerConn = net.Pipe()
			subPkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
			subPkg.Topics = []string{"/armada/" + deviceName + "/ota"}
			subPkg.Qoss = []byte{0}
			go func() {
				<-connected
				broker.Subscribe(subPkg, brokerConn)
			}()

			pkg, err := packets.ReadPacket(subscriberConn)
			Expect(err).NotTo(HaveOccurred())
			_, ok := pkg.(*packets.SubackPacket)
			Expect(ok).To(BeTrue())
		})
		It("publishes an error message", func() {
			deviceClient.Close()

			var ok bool
			var pubPkg *packets.PublishPacket
			//pubRead := make(chan bool)

			//go func() {
				pkg, err := packets.ReadPacket(subscriberConn)
				Expect(err).NotTo(HaveOccurred())
				pubPkg, ok = pkg.(*packets.PublishPacket)
				Expect(ok).To(BeTrue())
			//	pubRead <- true
			//}()

			//<-pubRead
			Expect(pubPkg.TopicName).To(Equal("/armada/" + deviceName + "/ota"))

			var otaState devices.OTAState
			err = json.Unmarshal(pubPkg.Payload, &otaState)
			Expect(err).NotTo(HaveOccurred())
			Expect(otaState.State).To(Equal(devices.Error))
		})
	})
})
