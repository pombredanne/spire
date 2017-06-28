package control_test

import (
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/control"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/handlers"
)

var _ = Describe("Control Message Handlers", func() {

	var devs *devices.DeviceMap

	Context("cloud to device", func() {
		var deviceClient, deviceServer, controlClient, controlServer net.Conn

		BeforeEach(func() {
			deviceClient, deviceServer = net.Pipe()
			controlServer, controlClient = net.Pipe()

			devs = devices.NewDeviceMap()
			devs.Add("1.marsara", deviceServer)

			ctrlMsgHandler := control.NewMessageHandler(devs)

			go func() {
				connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
				Expect(connPkg.Write(controlClient)).NotTo(HaveOccurred())

				// read CONNACK
				_, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())

				pubPkg := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
				pubPkg.TopicName = "/armada/1.marsara/foo"
				pubPkg.Payload = []byte(`{"foo": "bar"}`)
				Expect(pubPkg.Write(controlClient)).NotTo(HaveOccurred())
			}()

			go func() {
				ctrlMsgHandler.HandleConnection(controlServer)
			}()
		})
		It("forwards control messages", func() {
			rpkg, err := packets.ReadPacket(deviceClient)
			Expect(err).ToNot(HaveOccurred())

			pkg, ok := rpkg.(*packets.PublishPacket)
			Expect(ok).To(BeTrue())

			Expect(pkg.TopicName).To(Equal("/pylon/1.marsara/foo"))
			Expect(pkg.Payload).To(Equal([]byte(`{"foo": "bar"}`)))
		})
	})
	var devMsgHandler *handlers.DeviceMessageHandler
	var server net.Conn
	var client net.Conn
	var response packets.ControlPacket

	BeforeEach(func() {
		devs = devices.NewDeviceMap()
		devMsgHandler = handlers.NewDeviceMessageHandler(devs)

		server, client = net.Pipe()
		connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
		connPkg.ClientIdentifier = "1.marsara"

		go func() {
			Expect(connPkg.Write(client)).NotTo(HaveOccurred())
		}()

		go func() {
			devMsgHandler.HandleConnection(server)
		}()

		var err error
		response, err = packets.ReadPacket(client)
		Expect(err).NotTo(HaveOccurred())
	})
})
