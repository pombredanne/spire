package devices_test

import (
	"encoding/json"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

var _ = Describe("Device Message Handlers", func() {

	var devMsgHandler *devices.MessageHandler
	var deviceServer, deviceClient net.Conn
	var response packets.ControlPacket
	var broker *mqtt.Broker
	var handleConnectionReturned chan bool

	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"

	BeforeEach(func() {
		broker = mqtt.NewBroker()
		devMsgHandler = devices.NewMessageHandler(broker)
		deviceServer, deviceClient = net.Pipe()

		handleConnectionReturned = make(chan bool)
	})
	JustBeforeEach(func() {
		go func() {
			devMsgHandler.HandleConnection(deviceServer)
			handleConnectionReturned <- true
		}()

		connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
		connPkg.ClientIdentifier = deviceName
		connPkg.Username = formationID
		connPkg.UsernameFlag = true
		Expect(connPkg.Write(deviceClient)).NotTo(HaveOccurred())

		var err error
		response, err = packets.ReadPacket(deviceClient)
		Expect(err).NotTo(HaveOccurred())
	})
	Describe("connect", func() {
		It("sends CONNACK", func() {
			_, ok := response.(*packets.ConnackPacket)
			Expect(ok).To(BeTrue())
		})
		It("fetches device info and adds 'device_os' to device state", func() {
			deviceInfoState := devMsgHandler.GetDeviceState(formationID, deviceName, "device_info")
			Expect(deviceInfoState).NotTo(BeNil())
			Expect(deviceInfoState.(map[string]interface{})["device_os"]).To(Equal("tplink-archer-c7-lingrush-44"))
		})
		Describe("pub/sub", func() {
			var controlServer, controlClient net.Conn

			BeforeEach(func() {
				controlServer, controlClient = net.Pipe()
				pkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
				pkg.Topics = []string{"/armada/" + deviceName + "/up"}
				go broker.Subscribe(pkg, controlServer)

				// read suback
				_, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())
			})
			It("publishes an 'up' message for the device", func() {
				pkg, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())

				pubPkg, ok := pkg.(*packets.PublishPacket)
				Expect(ok).To(BeTrue())

				upState := make(map[string]interface{})
				err = json.Unmarshal(pubPkg.Payload, &upState)
				Expect(err).NotTo(HaveOccurred())
				Expect(upState["state"]).To(Equal("up"))
			})
		})
	})
	Describe("disconnect", func() {
		Context("by sending DISCONNECT", func() {
			var controlServer, controlClient net.Conn

			BeforeEach(func() {
				controlServer, controlClient = net.Pipe()
				pkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
				pkg.Topics = []string{"/armada/" + deviceName + "/up"}
				go broker.Subscribe(pkg, controlServer)

				// read suback
				_, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())
			})
			It("publishes an 'up' message for the device", func() {
				dPkg := packets.NewControlPacket(packets.Disconnect)
				Expect(dPkg.Write(deviceClient)).NotTo(HaveOccurred())

				// read and ignore the "up" state message that was published when the device connected
				_, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())

				pkg, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())

				pubPkg, ok := pkg.(*packets.PublishPacket)
				Expect(ok).To(BeTrue())

				upState := make(map[string]interface{})
				err = json.Unmarshal(pubPkg.Payload, &upState)
				Expect(err).NotTo(HaveOccurred())
				Expect(upState["state"]).To(Equal("down"))
			})
			Context("reconnect", func() {
				It("responds with CONNACK", func() {
					dPkg := packets.NewControlPacket(packets.Disconnect)
					Expect(dPkg.Write(deviceClient)).NotTo(HaveOccurred())

					deviceServer, deviceClient = net.Pipe()

					go func() {
						devMsgHandler.HandleConnection(deviceServer)
					}()

					connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
					connPkg.ClientIdentifier = deviceName
					connPkg.Username = formationID
					connPkg.UsernameFlag = true
					Expect(connPkg.Write(deviceClient)).NotTo(HaveOccurred())

					var err error
					response, err = packets.ReadPacket(deviceClient)
					Expect(err).NotTo(HaveOccurred())

					_, isConnAck := response.(*packets.ConnackPacket)
					Expect(isConnAck).To(BeTrue())
				})
			})
		})
		Context("by closing the connection", func() {
			var controlServer, controlClient net.Conn

			BeforeEach(func() {
				controlServer, controlClient = net.Pipe()
				pkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
				pkg.Topics = []string{"/armada/" + deviceName + "/up"}
				go broker.Subscribe(pkg, controlServer)

				// read suback
				_, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())
			})
			It("publishes an 'up' message for the device", func() {
				// read and ignore the "up" state message that was published when the device connected
				_, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())

				Expect(deviceClient.Close()).ToNot(HaveOccurred())
				<-handleConnectionReturned

				pkg, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())

				pubPkg, ok := pkg.(*packets.PublishPacket)
				Expect(ok).To(BeTrue())

				upState := make(map[string]interface{})
				err = json.Unmarshal(pubPkg.Payload, &upState)
				Expect(err).NotTo(HaveOccurred())
				Expect(upState["state"]).To(Equal("down"))
			})
		})
	})
})
