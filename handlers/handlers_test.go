package handlers_test

import (
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/handlers"
)

var _ = Describe("Device Message Handlers", func() {

	var devs *devices.DeviceMap
	var devMsgHandler *handlers.DeviceMessageHandler
	var server net.Conn
	var client net.Conn

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
	})

	It("sends CONNACK", func() {
		pkg, err := packets.ReadPacket(client)
		Expect(err).NotTo(HaveOccurred())
		_, ok := pkg.(*packets.ConnackPacket)
		Expect(ok).To(BeTrue())
	})
	It("adds the device", func() {
		_, err := packets.ReadPacket(client)
		Expect(err).NotTo(HaveOccurred())

		dev, err := devs.Get("1.marsara")
		Expect(err).NotTo(HaveOccurred())
		Expect(dev).NotTo(BeNil())
	})
})
