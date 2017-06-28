package devices_test

import (
	"net"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
)

var _ = Describe("Device Message Handlers", func() {

	var devs *devices.DeviceMap
	var devMsgHandler *devices.DeviceMessageHandler
	var server net.Conn
	var client net.Conn
	var response packets.ControlPacket

	BeforeEach(func() {
		devs = devices.NewDeviceMap()
		devMsgHandler = devices.NewDeviceMessageHandler(devs)
		server, client = net.Pipe()

		go func() {
			connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
			connPkg.ClientIdentifier = "1.marsara"
			Expect(connPkg.Write(client)).NotTo(HaveOccurred())
		}()

		go func() {
			devMsgHandler.HandleConnection(server)
		}()

		var err error
		response, err = packets.ReadPacket(client)
		Expect(err).NotTo(HaveOccurred())
	})
	It("sends CONNACK", func() {
		_, ok := response.(*packets.ConnackPacket)
		Expect(ok).To(BeTrue())
	})
	It("adds the device", func() {
		dev, err := devs.Get("1.marsara")
		Expect(err).NotTo(HaveOccurred())
		Expect(dev).NotTo(BeNil())
	})
	Context("disconnect", func() {
		Context("by sending DISCONNECT", func() {
			BeforeEach(func() {
				pkg := packets.NewControlPacket(packets.Disconnect)
				Expect(pkg.Write(client)).NotTo(HaveOccurred())
			})
			It("removes the device", func() {
				_, err := devs.Get("1.marsara")
				Expect(err).To(Equal(devices.ErrDevNotFound))
			})
		})
		Context("by closing the connection", func() {
			BeforeEach(func() {
				Expect(client.Close()).ToNot(HaveOccurred())
				buf := make([]byte, 5)
				server.Read(buf)
			})
			It("removes the device", func() {
				// Since the function under test runs in a goroutine, we need this to avoid a race condition. :(
				time.Sleep(time.Millisecond * 1)
				_, err := devs.Get("1.marsara")
				Expect(err).To(Equal(devices.ErrDevNotFound))
			})
		})
	})
})
