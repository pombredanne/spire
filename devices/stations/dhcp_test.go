package stations_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices/stations"
)

var _ = Describe("DHCP Parser", func() {

	var dhcpInput string
	var dhcpState stations.DHCPState
	var parseError error

	JustBeforeEach(func() {
		dhcpState, parseError = stations.ParseDHCP([]byte(dhcpInput))
		Expect(parseError).NotTo(HaveOccurred())
	})
	Describe("parses json messages", func() {
		BeforeEach(func() {
			dhcpInput = `{
				"wlan0": [
					{
						"m": "11:11:11:11:11:11",
						"ip": "192.168.1.100",
						"l": "4711",
						"n": "client1"
					},
					{
						"m": "22:22:22:22:22:22",
						"ip": "192.168.1.101",
						"l": "1337",
						"n": "client2"
					}
				],
				"wlan1": [
					{
						"m": "33:33:33:33:33:33",
						"ip": "192.168.1.102",
						"l": "2342",
						"n": "client3"
					}
				]
			}`
		})
		It("returns two entries", func() {
			Expect(len(dhcpState)).To(Equal(2))
		})
		It("includes wlan0 device", func() {
			dev0, exists := dhcpState["wlan0"]
			Expect(exists).To(BeTrue())
			Expect(len(dev0)).To(Equal(2))

			client1 := dev0[0]
			Expect(exists).To(BeTrue())
			Expect(client1.MAC).To(Equal("11:11:11:11:11:11"))
			Expect(client1.IP).To(Equal("192.168.1.100"))
			Expect(client1.Hostname).To(Equal("client1"))
			Expect(client1.TTL).To(Equal("4711"))

			client2 := dev0[1]
			Expect(exists).To(BeTrue())
			Expect(client2.MAC).To(Equal("22:22:22:22:22:22"))
			Expect(client2.IP).To(Equal("192.168.1.101"))
			Expect(client2.Hostname).To(Equal("client2"))
			Expect(client2.TTL).To(Equal("1337"))
		})
		It("includes wlan1 device", func() {
			dev1, exists := dhcpState["wlan1"]
			Expect(exists).To(BeTrue())
			Expect(len(dev1)).To(Equal(1))

			client3 := dev1[0]
			Expect(exists).To(BeTrue())
			Expect(client3.MAC).To(Equal("33:33:33:33:33:33"))
			Expect(client3.IP).To(Equal("192.168.1.102"))
			Expect(client3.Hostname).To(Equal("client3"))
			Expect(client3.TTL).To(Equal("2342"))
		})
	})
	Describe("parses legacy messages", func() {
		BeforeEach(func() {
			dhcpInput = "wlan0\n" +
				"11:11:11:11:11:11\t192.168.1.100\t4711\tclient1\n" +
				"22:22:22:22:22:22\t192.168.1.101\t1337\tclient2\n" +
				"wlan1\n" +
				"33:33:33:33:33:33\t192.168.1.102\t2342\tclient3\n"
		})
		It("returns two entries", func() {
			Expect(len(dhcpState)).To(Equal(2))
		})
		It("includes wlan0 device", func() {
			dev0, exists := dhcpState["wlan0"]
			Expect(exists).To(BeTrue())
			Expect(len(dev0)).To(Equal(2))

			client1 := dev0[0]
			Expect(exists).To(BeTrue())
			Expect(client1.MAC).To(Equal("11:11:11:11:11:11"))
			Expect(client1.IP).To(Equal("192.168.1.100"))
			Expect(client1.Hostname).To(Equal("client1"))
			Expect(client1.TTL).To(Equal("4711"))

			client2 := dev0[1]
			Expect(exists).To(BeTrue())
			Expect(client2.MAC).To(Equal("22:22:22:22:22:22"))
			Expect(client2.IP).To(Equal("192.168.1.101"))
			Expect(client2.Hostname).To(Equal("client2"))
			Expect(client2.TTL).To(Equal("1337"))
		})
		It("includes wlan1 device", func() {
			dev1, exists := dhcpState["wlan1"]
			Expect(exists).To(BeTrue())
			Expect(len(dev1)).To(Equal(1))

			client3 := dev1[0]
			Expect(exists).To(BeTrue())
			Expect(client3.MAC).To(Equal("33:33:33:33:33:33"))
			Expect(client3.IP).To(Equal("192.168.1.102"))
			Expect(client3.Hostname).To(Equal("client3"))
			Expect(client3.TTL).To(Equal("2342"))
		})
	})
	Describe("json marshaling", func() {
		var m map[string]string

		BeforeEach(func() {
			c := stations.DHCPClient{
				MAC:      "11:11:11:11:11:11",
				IP:       "1.2.3.4",
				Hostname: "bob",
				TTL:      "23",
			}

			buf, err := json.Marshal(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(buf, &m)).ToNot(HaveOccurred())
		})
		It("uses readable keys", func() {
			_, exists := m["m"]
			Expect(exists).To(BeFalse())

			Expect(m["mac"]).To(Equal("11:11:11:11:11:11"))
			Expect(m["ip"]).To(Equal("1.2.3.4"))
			Expect(m["host_name"]).To(Equal("bob"))
			Expect(m["ttl"]).To(Equal("23"))
		})
	})
})
