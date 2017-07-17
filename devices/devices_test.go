package devices_test

import (
	"net"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/service"
)

var _ = Describe("Device Message Handlers", func() {

	var state *devices.State
	var devs *devices.DeviceMap
	var devMsgHandler *devices.MessageHandler
	var server net.Conn
	var client net.Conn
	var response packets.ControlPacket
	var done chan bool

	BeforeEach(func() {
		state = devices.NewState()
		devs = state.Devices
		devMsgHandler = devices.NewMessageHandler(state, service.NewBroker())
		server, client = net.Pipe()

		go func() {
			connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
			connPkg.ClientIdentifier = "1.marsara"
			Expect(connPkg.Write(client)).NotTo(HaveOccurred())
		}()

		done = make(chan bool)
		go func() {
			devMsgHandler.HandleConnection(server)
			done <- true
		}()

		var err error
		response, err = packets.ReadPacket(client)
		Expect(err).NotTo(HaveOccurred())
	})
	Context("connect", func() {
		It("sends CONNACK", func() {
			_, ok := response.(*packets.ConnackPacket)
			Expect(ok).To(BeTrue())
		})
		It("adds the device", func() {
			time.Sleep(time.Microsecond * 10) // gross
			dev, err := devs.Get("1.marsara")
			Expect(err).NotTo(HaveOccurred())
			Expect(dev).NotTo(BeNil())
		})
		It("sets 'up' state on the device", func() {
			time.Sleep(time.Microsecond * 10) // gross
			dev, err := devs.Get("1.marsara")
			Expect(err).NotTo(HaveOccurred())

			upState, exists := dev.GetState("up")
			Expect(exists).To(BeTrue())
			Expect(upState.(map[string]interface{})["state"]).To(Equal("up"))
		})
	})
	Context("disconnect", func() {
		Context("by sending DISCONNECT", func() {
			BeforeEach(func() {
				pkg := packets.NewControlPacket(packets.Disconnect)
				Expect(pkg.Write(client)).NotTo(HaveOccurred())
			})
			It("updates 'up' state on the device", func() {
				dev, err := devs.Get("1.marsara")
				Expect(err).NotTo(HaveOccurred())

				upState, exists := dev.GetState("up")
				Expect(exists).To(BeTrue())
				Expect(upState.(map[string]interface{})["state"]).To(Equal("down"))
			})
			Context("reconnect", func() {
				BeforeEach(func() {
					devMsgHandler = devices.NewMessageHandler(state, service.NewBroker())
					server, client = net.Pipe()

					go func() {
						connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
						connPkg.ClientIdentifier = "1.marsara"
						Expect(connPkg.Write(client)).NotTo(HaveOccurred())
					}()

					done = make(chan bool)
					go func() {
						devMsgHandler.HandleConnection(server)
						done <- true
					}()

					var err error
					response, err = packets.ReadPacket(client)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
		Context("by closing the connection", func() {
			BeforeEach(func() {
				Expect(client.Close()).ToNot(HaveOccurred())
				<-done
			})
			It("updates 'up' state on the device", func() {
				dev, err := devs.Get("1.marsara")
				Expect(err).NotTo(HaveOccurred())

				upState, exists := dev.GetState("up")
				Expect(exists).To(BeTrue())
				Expect(upState.(map[string]interface{})["state"]).To(Equal("down"))
			})
		})
	})
})
