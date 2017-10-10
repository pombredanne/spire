package stargate_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/devices/stargate"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/testutils"
)

var _ = Describe("Stargate Message Handler", func() {

	var broker *mqtt.Broker
	var formations *devices.FormationMap
	var recorder *testutils.PubSubRecorder

	var deviceTopicPorts = "/pylon/1.marsara/stargate/port"
	var controlTopicPorts = "/matriarch/1.marsara/stargate/ports"
	var deviceTopicSystemImages = "/pylon/1.marsara/stargate/systemimaged"
	var controlTopicSystemImages = "/matriarch/1.marsara/stargate/system_images"
	var formationID = "00000000-0000-0000-0000-000000000000"
	var deviceName = "1.marsara"

	BeforeEach(func() {
		broker = mqtt.NewBroker()
		formations = devices.NewFormationMap()
		recorder = testutils.NewPubSubRecorder()

		stargate.Register(broker, formations)
		broker.Subscribe(controlTopicPorts, recorder.Record)
		broker.Subscribe(controlTopicSystemImages, recorder.Record)
	})
	Describe("handling port messages", func() {
		var state *stargate.State
		var portStateAfter *stargate.PortState
		var publishedPortMap stargate.PortMap
		var payload []byte

		var port = 1

		BeforeEach(func() {
			state = stargate.NewState()
		})
		JustBeforeEach(func() {
			formations.PutDeviceState(formationID, deviceName, stargate.Key, state)

			broker.Publish(deviceTopicPorts, payload)

			rawState := formations.GetDeviceState(deviceName, stargate.Key)
			Expect(rawState).NotTo(BeNil())

			var ok bool
			state, ok = rawState.(*stargate.State)
			Expect(ok).To(BeTrue())

			portStateAfter, ok = state.Ports[port]
			Expect(ok).To(BeTrue())

			Expect(recorder.Count()).To(BeNumerically("==", 1))

			topic, payload := recorder.First()
			Expect(topic).To(Equal(controlTopicPorts))

			publishedPortMap, ok = payload.(stargate.PortMap)
			Expect(ok).To(BeTrue())

			_, ok = publishedPortMap[port]
			Expect(ok).To(BeTrue())
		})
		Describe("'up' message", func() {
			BeforeEach(func() {
				payload = []byte(fmt.Sprintf(`{
					"port": %d,
					"up": true
				}`, port))
			})
			It("adds device state for port 1", func() {
				oneSecondAgo := time.Now().UTC().Add(time.Second * -1).Unix()
				Expect(portStateAfter.Timestamp).To(BeNumerically(">", oneSecondAgo))
			})
			It("publishes a message with port 1 in state 'wait'", func() {
				Expect(publishedPortMap[port].State).To(Equal(stargate.Wait))
			})
		})
		Describe("tftpd 'listening' message", func() {
			BeforeEach(func() {
				payload = []byte(fmt.Sprintf(`{
					"port": %d,
					"tftpd": {
						"listening": true
					}
				}`, port))
			})
			It("adds device state for port 1", func() {
				oneSecondAgo := time.Now().UTC().Add(time.Second * -1).Unix()
				Expect(portStateAfter.Timestamp).To(BeNumerically(">", oneSecondAgo))
			})
			It("publishes a message with port 1 in state 'wait'", func() {
				Expect(publishedPortMap[port].State).To(Equal(stargate.Wait))
			})
		})
		Describe("tftpd 'request' message", func() {
			BeforeEach(func() {
				state.Ports[port] = stargate.NewPortState()

				payload = []byte(fmt.Sprintf(`{
					"port": %d,
					"tftpd": {
						"request": "some file",
						"total": 4711
					}
				}`, port))
			})
			It("changes the state of port 1 to 'start'", func() {
				Expect(state.Ports[port].State).To(Equal(stargate.Start))
			})
			It("publishes a message with 'file' and 'total'", func() {
				p := publishedPortMap[port]
				Expect(p.File).To(Equal("some file"))
				Expect(p.Total).To(BeNumerically("==", 4711))
			})
		})
		Describe("tftpd 'transmitting' message", func() {
			BeforeEach(func() {
				ps := stargate.NewPortState()
				ps.Total = 4711
				state.Ports[port] = ps

				payload = []byte(fmt.Sprintf(`{
					"port": %d,
					"tftpd": {
						"transmitting": 2300
					}
				}`, port))
			})
			It("publishes a message with progress in percent", func() {
				p := publishedPortMap[port]
				Expect(p.State).To(Equal(stargate.Transmitting))
				Expect(p.Progress).To(BeNumerically(">", 48))
				Expect(p.Progress).To(BeNumerically("<", 50))
			})
		})
		Describe("tftpd 'finished' message", func() {
			BeforeEach(func() {
				state.Ports[port] = stargate.NewPortState()

				payload = []byte(fmt.Sprintf(`{
					"port": %d,
					"tftpd": {
						"finished": true
					}
				}`, port))
			})
			It("publishes a message with the state of port 1 set to 'flashing'", func() {
				p := publishedPortMap[port]
				Expect(p.State).To(Equal(stargate.Flashing))
				Expect(p.Progress).To(BeNumerically("==", 100))
			})
		})
		Describe("tftpd 'error' message", func() {
			BeforeEach(func() {
				state.Ports[port] = stargate.NewPortState()

				payload = []byte(fmt.Sprintf(`{
					"port": %d,
					"tftpd": {
						"error": "the dog ate my homework"
					}
				}`, port))
			})
			It("publishes a message with the state of port 1 set to 'error'", func() {
				p := publishedPortMap[port]
				Expect(p.State).To(Equal(stargate.Error))
				Expect(p.Error).To(Equal("the dog ate my homework"))
			})
		})
		Describe("'assimilator' message", func() {
			BeforeEach(func() {
				state.Ports[port] = stargate.NewPortState()

				payload = []byte(fmt.Sprintf(`{
					"port": %d,
					"assimilator": {
						"message": {
							"assimilator": "oh wow, such nesting, many json"
						},
						"from": {
							"ip": "1.2.3.4.5",
							"port": 1337
						}
					}
				}`, port))
			})
			It("publishes a message with whatever assimilator sent", func() {
				p := publishedPortMap[port]
				Expect(p.State).To(Equal(stargate.Assimilator))
				Expect(p.Assimilator).To(Equal("oh wow, such nesting, many json"))
			})
		})
	})
	Describe("handling system image messages", func() {
		var state *stargate.State
		var publishedSysImgState *stargate.SystemImageState
		var payload []byte
		var sysImgID = "image1"

		BeforeEach(func() {
			state = stargate.NewState()
		})
		JustBeforeEach(func() {
			formations.PutDeviceState(formationID, deviceName, stargate.Key, state)
			broker.Publish(deviceTopicSystemImages, payload)
			Expect(recorder.Count()).To(BeNumerically("==", 1))

			topic, payload := recorder.First()
			Expect(topic).To(Equal(controlTopicSystemImages))

			sysImgMap, ok := payload.(stargate.SystemImageMap)
			Expect(ok).To(BeTrue())

			publishedSysImgState, ok = sysImgMap[sysImgID]
			Expect(ok).To(BeTrue())
		})
		Describe("'api' message", func() {
			BeforeEach(func() {
				payload = []byte(fmt.Sprintf(`{
					"id": "%s",
					"download": "api",
					"vendor": "acme",
					"product": "wifi"
				}`, sysImgID))
			})
			It("adds device state for image1", func() {
				Expect(publishedSysImgState).NotTo(BeNil())
			})
			It("publishes a message with image1 in state 'start'", func() {
				Expect(publishedSysImgState.State).To(Equal(stargate.Start))
			})
		})
		Describe("subsequent system image messages", func() {
			BeforeEach(func() {
				state.SystemImages[sysImgID] = stargate.NewSystemImageState(sysImgID, "acme", "wifi")
			})
			Describe("'start' message", func() {
				BeforeEach(func() {
					payload = []byte(fmt.Sprintf(`{
						"id": "%s",
						"download": "start",
						"total": 4711
					}`, sysImgID))
				})
				It("publishes a message with image1 in state 'start'", func() {
					Expect(publishedSysImgState.State).To(Equal(stargate.Start))
					Expect(publishedSysImgState.Total).To(BeNumerically("==", 4711))
				})
			})
			Describe("'progress' message", func() {
				BeforeEach(func() {
					state.SystemImages[sysImgID].Total = 4000

					payload = []byte(fmt.Sprintf(`{
						"id": "%s",
						"download": "progress",
						"progress": 2000
					}`, sysImgID))
				})
				It("publishes a message with download progress", func() {
					Expect(publishedSysImgState.Progress).To(BeNumerically("==", 50))
				})
			})
			Describe("'ok' message", func() {
				BeforeEach(func() {
					state.SystemImages[sysImgID].Total = 4000

					payload = []byte(fmt.Sprintf(`{
						"id": "%s",
						"download": "ok"
					}`, sysImgID))
				})
				It("publishes a message with image1 in state 'ok'", func() {
					Expect(publishedSysImgState.State).To(Equal(stargate.Ok))
					Expect(publishedSysImgState.Progress).To(BeNumerically("==", 100))
				})
			})
			Describe("'error' message", func() {
				BeforeEach(func() {
					state.SystemImages[sysImgID].Total = 4000

					payload = []byte(fmt.Sprintf(`{
						"id": "%s",
						"download": "error"
					}`, sysImgID))
				})
				It("publishes a message with image1 in state 'error'", func() {
					Expect(publishedSysImgState.State).To(Equal(stargate.Error))
				})
			})
		})
	})
})
