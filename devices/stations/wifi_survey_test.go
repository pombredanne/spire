package stations_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices/stations"
)

var _ = Describe("Wifi Survey Parser", func() {

	var surveyInput string
	var surveys map[string]*stations.WifiSurvey
	var parseError error

	JustBeforeEach(func() {
		surveys, parseError = stations.ParseWifiSurvey(surveyInput)
	})
	Describe("parses emtpy string", func() {
		BeforeEach(func() {
			surveyInput = ""
		})
		It("returns an empty survey", func() {
			Expect(parseError).NotTo(HaveOccurred())
			Expect(surveys).NotTo(BeNil())
			Expect(surveys).To(BeEmpty())
		})
	})
	Describe("parses a valid survey", func() {
		BeforeEach(func() {
			surveyInput = `
Survey data from wlan-private-g
  frequency:      2412 MHz
  noise:        -95 dBm
  channel active time:    59 ms
  channel busy time:    32 ms
  channel receive time:   29 ms
  channel transmit time:    2 ms
Survey data from wlan-private-g
  frequency:      2417 MHz
  noise:        -95 dBm
  channel active time:    59 ms
  channel busy time:    44 ms
  channel receive time:   27 ms
  channel transmit time:    3 ms
Survey data from wlan-private-g
  frequency:      2422 MHz
  noise:        -95 dBm
  channel active time:    59 ms
  channel busy time:    22 ms
  channel receive time:   16 ms
  channel transmit time:    2 ms
Survey data from wlan-private-g           <- double with first entry, but has more active time!
  frequency:      2412 MHz
  noise:        -95 dBm
  channel active time:    70039858 ms
  channel busy time:    36751702 ms
  channel receive time:   33080329 ms
  channel transmit time:    2157771 ms
Survey data from wlan-private-a
        frequency:                      5180 MHz [in use]
        noise:                          -106 dBm
        channel active time:            635043 ms
        channel busy time:              137081 ms
        channel receive time:           89174 ms
        channel transmit time:          864 ms
Survey data from wlan-private-a
        frequency:                      5200 MHz
        noise:                          -108 dBm
        channel active time:            146 ms
        channel busy time:              16 ms
Survey data from wlan-private-a
        frequency:                      5220 MHz
Survey data from wlan-private-a
        frequency:                      5240 MHz
Survey data from wlan-private-a
        frequency:                      5260 MHz
`
		})
		It("returns 8 entries", func() {
			Expect(len(surveys)).To(Equal(8))
		})
		It("parses 'channel busy time'", func() {
			s := surveys["5200 MHz"]
			Expect(s).NotTo(BeNil())
			Expect(s.ChannelBusyTime).To(Equal("16 ms"))
		})
		It("sets the 'in use' flag", func() {
			s := surveys["5180 MHz"]
			Expect(s).NotTo(BeNil())
			Expect(s.InUse).To(BeTrue())
		})
		It("includes info from the survey with longer 'channel active time'", func() {
			s := surveys["2412 MHz"]
			Expect(s).NotTo(BeNil())
			Expect(s.ChannelActiveTime).To(Equal("70039858 ms"))
		})
	})
})
