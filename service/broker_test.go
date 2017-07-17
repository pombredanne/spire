package service_test

import (
	"encoding/json"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	devpkg "github.com/superscale/spire/devices"
	"github.com/superscale/spire/service"
)

var _ = Describe("Broker", func() {
	var brokerConn, subscriberConn net.Conn
	var broker *service.Broker

	Context("subscribe", func() {
		var subResponse packets.ControlPacket

		BeforeEach(func() {
			brokerConn, subscriberConn = net.Pipe()
			broker = service.NewBroker(devpkg.NewDeviceMap())

			subPkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
			subPkg.Topics = []string{"/pylon/1.marsara/up"}
			subPkg.Qoss = []byte{0}
			subPkg.MessageID = 1337

			go broker.Subscribe(subPkg, brokerConn)

			var err error
			subResponse, err = packets.ReadPacket(subscriberConn)
			Expect(err).NotTo(HaveOccurred())
		})
		It("responds with SUBACK", func() {
			subAckPkg, isSubAck := subResponse.(*packets.SubackPacket)
			Expect(isSubAck).To(BeTrue())
			Expect(subAckPkg.MessageID).To(Equal(uint16(1337)))
		})
		Context("publish", func() {
			var pkg packets.ControlPacket

			BeforeEach(func() {
				payload := map[string]string{"foo": "bar"}
				pubPkg, err := service.MakePublishPacket("/pylon/1.marsara/up", payload)
				Expect(err).NotTo(HaveOccurred())

				broker.Publish(pubPkg)

				pkg, err = packets.ReadPacket(subscriberConn)
				Expect(err).NotTo(HaveOccurred())
			})
			It("forwards the message to subscribers", func() {
				pubPkg, ok := pkg.(*packets.PublishPacket)
				Expect(ok).To(BeTrue())

				Expect(pubPkg.TopicName).To(Equal("/pylon/1.marsara/up"))

				var payload map[string]string
				err := json.Unmarshal(pubPkg.Payload, &payload)
				Expect(err).NotTo(HaveOccurred())
				Expect(payload["foo"]).To(Equal("bar"))
			})
		})
		Context("unsubscribe", func() {
			var response packets.ControlPacket

			BeforeEach(func() {
				unsubPkg := packets.NewControlPacket(packets.Unsubscribe).(*packets.UnsubscribePacket)
				unsubPkg.Topics = []string{"/pylon/1.marsara/up"}
				unsubPkg.MessageID = 1338

				go broker.Unsubscribe(unsubPkg, brokerConn)

				var err error
				response, err = packets.ReadPacket(subscriberConn)
				Expect(err).NotTo(HaveOccurred())
			})
			It("responds with UNSUBACK", func() {
				unsubAckPkg, ok := response.(*packets.UnsubackPacket)
				Expect(ok).To(BeTrue())
				Expect(unsubAckPkg.MessageID).To(Equal(uint16(1338)))
			})
		})
	})
})
