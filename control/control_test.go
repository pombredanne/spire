package control_test

import (
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/control"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/service"
)

var _ = Describe("Control Message Handlers", func() {

	var state *devices.State
	var devs *devices.DeviceMap
	var deviceClient, deviceServer, controlClient, controlServer net.Conn

	BeforeEach(func() {
		deviceClient, deviceServer = net.Pipe()
		controlServer, controlClient = net.Pipe()

		state = devices.NewState()
		devs = state.Devices
		devs.Add("1.marsara", deviceServer)
	})

	Context("cloud to device", func() {
		BeforeEach(func() {
			ctrlMsgHandler := control.NewMessageHandler(state, service.NewBroker())

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

	Context("device to cloud", func() {
		var ctrlMsgHandler *control.MessageHandler
		var response packets.ControlPacket
		var done chan bool

		BeforeEach(func() {
			ctrlMsgHandler = control.NewMessageHandler(state, service.NewBroker())
			done = make(chan bool)

			go func() {
				connPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
				Expect(connPkg.Write(controlClient)).NotTo(HaveOccurred())

				// read CONNACK
				_, err := packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())

				subPkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
				subPkg.Topics = []string{"/matriarch/1.marsara/#"}
				subPkg.Qoss = []byte{0}
				Expect(subPkg.Write(controlClient)).NotTo(HaveOccurred())

				response, err = packets.ReadPacket(controlClient)
				Expect(err).NotTo(HaveOccurred())
				done <- true
			}()

			go func() {
				ctrlMsgHandler.HandleConnection(controlServer)
			}()
		})
		It("responds with SUBACK", func() {
			<-done
			_, isSubAck := response.(*packets.SubackPacket)
			Expect(isSubAck).To(BeTrue())
		})
	})
})
