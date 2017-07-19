package devices_test

import (
	"net"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

var _ = Describe("Device Message Handlers", func() {

	var devMsgHandler *devices.MessageHandler
	var server net.Conn
	var client net.Conn
	var response packets.ControlPacket
	var done chan bool

	var formationID = "00000000-0000-0000-0000-000000000001"
	var deviceName = "1.marsara"

	BeforeEach(func() {
		devMsgHandler = devices.NewMessageHandler(mqtt.NewBroker())
		server, client = net.Pipe()

		done = make(chan bool)
		go func() {
			devMsgHandler.HandleConnection(server)
			done <- true
		}()

		connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
		connPkg.ClientIdentifier = deviceName
		connPkg.Username = formationID
		connPkg.UsernameFlag = true
		Expect(connPkg.Write(client)).NotTo(HaveOccurred())

		var err error
		response, err = packets.ReadPacket(client)
		Expect(err).NotTo(HaveOccurred())
	})
	Context("connect", func() {
		It("sends CONNACK", func() {
			_, ok := response.(*packets.ConnackPacket)
			Expect(ok).To(BeTrue())
		})
		It("sets 'up' state on the device", func() {
			time.Sleep(time.Millisecond * 1) // gross
			upState := devMsgHandler.GetDeviceState(formationID, deviceName, "up")
			Expect(upState).NotTo(BeNil())
			Expect(upState.(map[string]interface{})["state"]).To(Equal("up"))
		})
	})
	Context("disconnect", func() {
		Context("by sending DISCONNECT", func() {
			BeforeEach(func() {
				pkg := packets.NewControlPacket(packets.Disconnect)
				Expect(pkg.Write(client)).NotTo(HaveOccurred())
				<-done
			})
			It("updates 'up' state on the device", func() {
				upState := devMsgHandler.GetDeviceState(formationID, deviceName, "up")
				Expect(upState).NotTo(BeNil())
				Expect(upState.(map[string]interface{})["state"]).To(Equal("down"))
			})
			Context("reconnect", func() {
				BeforeEach(func() {
					server, client = net.Pipe()

					go func() {
						devMsgHandler.HandleConnection(server)
						done <- true
					}()

					go func() {
						connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
						connPkg.ClientIdentifier = deviceName
						connPkg.Username = formationID
						connPkg.UsernameFlag = true
						Expect(connPkg.Write(client)).NotTo(HaveOccurred())
					}()

					var err error
					response, err = packets.ReadPacket(client)
					Expect(err).NotTo(HaveOccurred())
				})
				It("responds with CONNACK", func() {
					_, isConnAck := response.(*packets.ConnackPacket)
					Expect(isConnAck).To(BeTrue())
				})
			})
		})
		Context("by closing the connection", func() {
			BeforeEach(func() {
				Expect(client.Close()).ToNot(HaveOccurred())
				<-done
			})
			It("updates 'up' state on the device", func() {
				upState := devMsgHandler.GetDeviceState(formationID, deviceName, "up")
				Expect(upState).NotTo(BeNil())
				Expect(upState.(map[string]interface{})["state"]).To(Equal("down"))
			})
		})
	})
})
