package stations_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices/stations"
)

var _ = Describe("Wifi Stations Parser", func() {

	var stationsInput = `
Station 4C:7C:5F:FF:FF:FF (on wlan-private-a)
      inactive time:  370 ms
      rx bytes:       55456
      rx packets:     814
      tx bytes:       36043
      tx packets:     260
      tx retries:     0
      tx failed:      2
      signal:         -45 dBm
      signal avg:     -46 dBm
      tx bitrate:     6.0 MBit/s
      rx bitrate:     24.0 MBit/s
      authorized:     yes
      authenticated:  yes
      preamble:       long
      WMM/WME:        yes
      MFP:            no
      TDLS peer:      no
      connected time: 162 seconds
Station 4C:7C:5F:FE:FE:FE (on wlan-private-a)
      inactive time:  23 seconds
      rx bytes:       4711
      rx packets:     814
      tx bytes:       36043
      tx packets:     260
      tx retries:     0
      tx failed:      2
      signal:         -45 dBm
      signal avg:     -46 dBm
      tx bitrate:     6.0 MBit/s
      rx bitrate:     24.0 MBit/s
      authorized:     yes
      authenticated:  yes
      preamble:       long
      WMM/WME:        yes
      MFP:            no
      TDLS peer:      no
      connected time: 162 seconds
`
	var wifiStations map[string]stations.WifiStation
	var parseError error

	JustBeforeEach(func() {
		wifiStations, parseError = stations.ParseWifiStations(stationsInput, "wlan-private-a")
		Expect(parseError).NotTo(HaveOccurred())
	})
	It("each station has 22 attributes", func() {
		Expect(len(wifiStations["4C:7C:5F:FF:FF:FF"])).To(Equal(22))
		Expect(len(wifiStations["4C:7C:5F:FE:FE:FE"])).To(Equal(22))
	})
	It("station data includes 'rx bytes'", func() {
		s1 := wifiStations["4C:7C:5F:FF:FF:FF"]
		s2 := wifiStations["4C:7C:5F:FE:FE:FE"]
		Expect(s1["rx bytes"]).To(Equal("55456"))
		Expect(s2["rx bytes"]).To(Equal("4711"))
	})
	It("normalizes 'inactive_time'", func() {
		s1 := wifiStations["4C:7C:5F:FF:FF:FF"]
		s2 := wifiStations["4C:7C:5F:FE:FE:FE"]
		Expect(s1["inactive_time"]).To(Equal(0))
		Expect(s1["inactive time"]).To(BeNil())
		Expect(s2["inactive_time"]).To(Equal(23))
		Expect(s2["inactive time"]).To(BeNil())
	})
	It("adds 'mode' and 'radio' from the interface name", func() {
		s := wifiStations["4C:7C:5F:FF:FF:FF"]
		Expect(s["mode"]).To(Equal("private"))
		Expect(s["radio"]).To(Equal("a"))
	})
})
