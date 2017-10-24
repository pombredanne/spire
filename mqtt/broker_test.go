package mqtt_test

import (
	"encoding/json"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/testutils"
)

var _ = Describe("Broker", func() {

	var brokerSession, subscriberSession *mqtt.Session
	var broker *mqtt.Broker

	BeforeEach(func() {
		brokerSession, subscriberSession = testutils.Pipe()
		broker = mqtt.NewBroker(false)
	})
	Context("pingreq", func() {
		var response packets.ControlPacket

		BeforeEach(func() {
			go broker.HandleConnection(brokerSession)

			subscriberSession.Write(packets.NewControlPacket(packets.Connect))
			_, err := subscriberSession.Read()
			Expect(err).NotTo(HaveOccurred())

			pkg := packets.NewControlPacket(packets.Pingreq)
			err = subscriberSession.Write(pkg)
			Expect(err).NotTo(HaveOccurred())

			response, err = subscriberSession.Read()
			Expect(err).NotTo(HaveOccurred())
		})
		It("responds with pingresp", func() {
			_, ok := response.(*packets.PingrespPacket)
			Expect(ok).To(BeTrue())
		})
	})
	Context("subscribe", func() {
		BeforeEach(func() {
			subPkg := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
			subPkg.Topics = []string{"pylon/1.marsara/up"}
			subPkg.MessageID = 1337

			broker.SubscribeAll(subPkg, brokerSession)
		})
		AfterEach(func() {
			brokerSession.Close()
		})
		Context("publish", func() {
			var pkg packets.ControlPacket

			BeforeEach(func() {
				go broker.Publish("pylon/1.marsara/up", map[string]string{"foo": "bar"})

				var err error
				pkg, err = subscriberSession.Read()
				Expect(err).NotTo(HaveOccurred())
			})
			It("forwards the message to subscribers", func() {
				pubPkg, ok := pkg.(*packets.PublishPacket)
				Expect(ok).To(BeTrue())

				Expect(pubPkg.TopicName).To(Equal("pylon/1.marsara/up"))

				var payload map[string]string
				err := json.Unmarshal(pubPkg.Payload, &payload)
				Expect(err).NotTo(HaveOccurred())
				Expect(payload["foo"]).To(Equal("bar"))
			})
		})
		Context("publish to a non-matching topic", func() {
			BeforeEach(func() {
				broker.Publish("pylon/2.korhal/up", map[string]string{"foo": "bar"})

				go func() {
					time.Sleep(time.Millisecond * 1)
					brokerSession.Close()
				}()
			})
			It("does not forward the message", func() {
				pkg, err := subscriberSession.Read()
				Expect(err).To(HaveOccurred())
				Expect(pkg).To(BeNil())
			})
		})
		Context("publish to a zero-length topic", func() {
			var recorder *testutils.PubSubRecorder

			BeforeEach(func() {
				recorder = testutils.NewPubSubRecorder()
				broker.Subscribe("", recorder)
				broker.Publish("", map[string]string{"foo": "bar"})
			})
			It("does not forward the message", func() {
				Expect(recorder.Count()).To(BeZero())
			})
		})
		Context("unsubscribe", func() {
			BeforeEach(func() {
				unsubPkg := packets.NewControlPacket(packets.Unsubscribe).(*packets.UnsubscribePacket)
				unsubPkg.Topics = []string{"pylon/1.marsara/up"}
				unsubPkg.MessageID = 1338

				broker.UnsubscribeAll(unsubPkg, brokerSession)

				broker.Publish("pylon/2.marsara/up", map[string]string{"foo": "bar"})

				go func() {
					time.Sleep(time.Millisecond * 1)
					brokerSession.Close()
				}()
			})
			It("does not forward the message", func() {
				pkg, err := subscriberSession.Read()
				Expect(err).To(HaveOccurred())
				Expect(pkg).To(BeNil())
			})
		})
	})
	Context("pub/sub with topicPrefix enabled", func() {
		var subTopic, pubTopic string
		var recorder *testutils.PubSubRecorder

		JustBeforeEach(func() {
			broker = mqtt.NewBroker(true)
			recorder = testutils.NewPubSubRecorder()
			broker.Subscribe(subTopic, recorder)
			broker.Publish(pubTopic, map[string]string{"foo": "bar"})
		})
		Context("subscribe without leading slash", func() {
			BeforeEach(func() {
				subTopic = "pylon/1.marsara/up"
				pubTopic = "/pylon/1.marsara/up"
			})
			It("forwards the message", func() {
				Expect(recorder.Count()).NotTo(BeZero())

				topic, payload := recorder.First()
				Expect(topic).To(Equal("/pylon/1.marsara/up"))

				msg, ok := payload.(map[string]string)
				Expect(ok).To(BeTrue())
				Expect(msg["foo"]).To(Equal("bar"))
			})
		})
		Context("publish without leading slash", func() {
			BeforeEach(func() {
				subTopic = "/pylon/1.marsara/up"
				pubTopic = "pylon/1.marsara/up"
			})
			It("forwards the message", func() {
				Expect(recorder.Count()).NotTo(BeZero())

				topic, payload := recorder.First()
				Expect(topic).To(Equal("/pylon/1.marsara/up"))

				msg, ok := payload.(map[string]string)
				Expect(ok).To(BeTrue())
				Expect(msg["foo"]).To(Equal("bar"))
			})
		})
	})
	Context("topic matching", func() {
		var publishTopic string
		var topics []string
		var matches []string

		JustBeforeEach(func() {
			matches = mqtt.MatchTopics(publishTopic, topics)
		})
		Context("exact match", func() {
			BeforeEach(func() {
				publishTopic = "armada/1.marsara/up"

				topics = []string{
					"armada/2.zenn/stations",
					"armada/1.marsara/ota",
					"armada/1.marsara/up",
					"pylon/1.marsara/up",
				}
			})
			It("returns the matching topic", func() {
				Expect(len(matches)).To(Equal(1))
				Expect(matches[0]).To(Equal("armada/1.marsara/up"))
			})
		})
		Context("with multi-level wildcard '#' at the end", func() {
			BeforeEach(func() {
				publishTopic = "armada/1.marsara/up"

				topics = []string{
					"armada/2.zenn/stations",
					"armada/1.marsara/ota",
					"armada/1.marsara/#",
					"armada/3.korhal/#",
				}
			})
			It("matches", func() {
				Expect(len(matches)).To(Equal(1))
				Expect(matches[0]).To(Equal("armada/1.marsara/#"))
			})
		})
		Context("with multi-level wildcards in the middle", func() {
			BeforeEach(func() {
				publishTopic = "armada/1.marsara/up"

				topics = []string{
					"armada/#/up",
				}
			})
			It("does not match", func() {
				Expect(len(matches)).To(Equal(0))
			})
		})
		Context("with single-level wildcards", func() {
			BeforeEach(func() {
				publishTopic = "armada/1.marsara/sys/facts"

				topics = []string{
					"armada/2.zenn/stations",
					"armada/1.marsara/ota",
					"armada/1.marsara/+",
					"armada/+/sys/facts",
					"armada/+/sys/#",
				}
			})
			It("returns the matching topic", func() {
				Expect(len(matches)).To(Equal(2))
				Expect(matches[0]).To(Equal("armada/+/sys/facts"))
				Expect(matches[1]).To(Equal("armada/+/sys/#"))
			})
		})
	})
	Context("multiple subscribers", func() {
		var sub1, sub2 *testutils.PubSubRecorder
		var topic = "foo/bar"
		var payload = "hi"

		BeforeEach(func() {
			sub1, sub2 = testutils.NewPubSubRecorder(), testutils.NewPubSubRecorder()

			broker.Subscribe(topic, sub1)
			broker.Subscribe(topic, sub2)

			broker.Publish(topic, payload)
		})
		It("publishes the message to all subscribers", func() {
			Eventually(func() int {
				return sub1.Count()
			}).Should(BeNumerically("==", 1))

			Eventually(func() int {
				return sub2.Count()
			}).Should(BeNumerically("==", 1))

			t, msg := sub1.First()
			Expect(t).To(Equal(topic))
			Expect(msg).To(Equal(payload))

			t, msg = sub2.First()
			Expect(t).To(Equal(topic))
			Expect(msg).To(Equal(payload))
		})
	})
	Describe("internal topics", func() {
		var sub *testutils.PubSubRecorder
		var internalTopic = mqtt.InternalTopicPrefix + "/foo/bar"
		var regularTopic = "foo/bar"
		var payload = "hi"

		BeforeEach(func() {
			sub = testutils.NewPubSubRecorder()

			broker.Subscribe(internalTopic, sub)
			broker.Subscribe(regularTopic, sub)

			go broker.HandleConnection(brokerSession)

			go func() {
				conPkg := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
				Expect(subscriberSession.Write(conPkg)).NotTo(HaveOccurred())

				_, err := subscriberSession.Read()
				Expect(err).NotTo(HaveOccurred())

				subscriberSession.HandleMessage(internalTopic, payload)
				subscriberSession.HandleMessage(regularTopic, payload)
			}()
		})
		It("ignores messages published by clients to internal topics", func() {
			Eventually(func() int {
				return sub.Count()
			}).Should(BeNumerically("==", 1))

			topic, _ := sub.First()
			Expect(topic).To(Equal(regularTopic))
		})
	})
})
