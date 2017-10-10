package stations_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices/stations"
)

var _ = Describe("Switch Parser", func() {

	var switchInput string
	var cpuPorts []string
	var ports stations.SwitchState
	var macs map[string]string
	var parseError error

	BeforeEach(func() {
		cpuPorts = []string{}
	})
	JustBeforeEach(func() {
		ports, macs, parseError = stations.ParseSwitch(switchInput, cpuPorts...)
		Expect(parseError).NotTo(HaveOccurred())
	})
	Describe("parses switch messages", func() {
		BeforeEach(func() {
			switchInput = `Global attributes:
	enable_vlan: 1
	enable_mirror_rx: 0
	enable_mirror_tx: 0
	mirror_monitor_port: 0
	mirror_source_port: 0
	arl_age_time: 300
	arl_table: address resolution table
Port 0: MAC aa:aa:aa:aa:aa:aa
Port 1: MAC bb:bb:bb:bb:bb:bb
Port 1: MAC cc:cc:cc:cc:cc:cc

	igmp_snooping: 0
	igmp_v3: 0
Port 0:
	mib: Port 0 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 25923
TxMulti    : 14
TxBroad    : 11
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 111
Tx65Byte   : 20742
Tx128Byte  : 1293
Tx256Byte  : 1587
Tx512Byte  : 609
Tx1024Byte : 1606
TxByte     : 5119287
RxDrop     : 0
RxFiltered : 0
RxUni      : 128705
RxMulti    : 0
RxBroad    : 3
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 54
Rx65Byte   : 8573
Rx128Byte  : 1535
Rx256Byte  : 285
Rx512Byte  : 306
Rx1024Byte : 117955
RxByte     : 180182272
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 5
	link: port:0 link:up speed:1000baseT full-duplex
Port 1:
	mib: Port 1 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 0
TxMulti    : 7
TxBroad    : 0
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 0
Tx65Byte   : 5
Tx128Byte  : 2
Tx256Byte  : 0
Tx512Byte  : 0
Tx1024Byte : 0
TxByte     : 766
RxDrop     : 0
RxFiltered : 0
RxUni      : 0
RxMulti    : 74
RxBroad    : 6
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 16
Rx65Byte   : 33
Rx128Byte  : 2
Rx256Byte  : 29
Rx512Byte  : 0
Rx1024Byte : 0
RxByte     : 14434
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 1
	link: port:1 link:up speed:1000baseT full-duplex
Port 2:
	mib: Port 2 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 0
TxMulti    : 0
TxBroad    : 0
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 0
Tx65Byte   : 0
Tx128Byte  : 0
Tx256Byte  : 0
Tx512Byte  : 0
Tx1024Byte : 0
TxByte     : 0
RxDrop     : 0
RxFiltered : 0
RxUni      : 0
RxMulti    : 0
RxBroad    : 0
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 0
Rx65Byte   : 0
Rx128Byte  : 0
Rx256Byte  : 0
Rx512Byte  : 0
Rx1024Byte : 0
RxByte     : 0
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 2
	link: port:2 link:down
Port 3:
	mib: Port 3 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 0
TxMulti    : 0
TxBroad    : 0
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 0
Tx65Byte   : 0
Tx128Byte  : 0
Tx256Byte  : 0
Tx512Byte  : 0
Tx1024Byte : 0
TxByte     : 0
RxDrop     : 0
RxFiltered : 0
RxUni      : 0
RxMulti    : 0
RxBroad    : 0
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 0
Rx65Byte   : 0
Rx128Byte  : 0
Rx256Byte  : 0
Rx512Byte  : 0
Rx1024Byte : 0
RxByte     : 0
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 3
	link: port:3 link:down
Port 4:
	mib: Port 4 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 0
TxMulti    : 0
TxBroad    : 0
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 0
Tx65Byte   : 0
Tx128Byte  : 0
Tx256Byte  : 0
Tx512Byte  : 0
Tx1024Byte : 0
TxByte     : 0
RxDrop     : 0
RxFiltered : 0
RxUni      : 0
RxMulti    : 0
RxBroad    : 0
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 0
Rx65Byte   : 0
Rx128Byte  : 0
Rx256Byte  : 0
Rx512Byte  : 0
Rx1024Byte : 0
RxByte     : 0
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 4
	link: port:4 link:down
Port 5:
	mib: Port 5 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 0
TxMulti    : 0
TxBroad    : 0
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 0
Tx65Byte   : 0
Tx128Byte  : 0
Tx256Byte  : 0
Tx512Byte  : 0
Tx1024Byte : 0
TxByte     : 0
RxDrop     : 0
RxFiltered : 0
RxUni      : 0
RxMulti    : 0
RxBroad    : 0
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 0
Rx65Byte   : 0
Rx128Byte  : 0
Rx256Byte  : 0
Rx512Byte  : 0
Rx1024Byte : 0
RxByte     : 0
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 0
	link: port:5 link:down
Port 6:
	mib: Port 6 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 128705
TxMulti    : 74
TxBroad    : 9
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 0
Tx65Byte   : 8676
Tx128Byte  : 1537
Tx256Byte  : 314
Tx512Byte  : 306
Tx1024Byte : 117955
TxByte     : 180711858
RxDrop     : 0
RxFiltered : 66
RxUni      : 25923
RxMulti    : 86
RxBroad    : 12
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 111
Rx65Byte   : 20791
Rx128Byte  : 1288
Rx256Byte  : 1612
Rx512Byte  : 612
Rx1024Byte : 1607
RxByte     : 5231786
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 0
	link: port:6 link:up speed:1000baseT full-duplex
Port 7:
	mib: Port 7 MIB counters
TxDrop     : 0
TxCRC      : 0
TxUni      : 0
TxMulti    : 0
TxBroad    : 0
TxCollision: 0
TxSingleCol: 0
TxMultiCol : 0
TxDefer    : 0
TxLateCol  : 0
TxExcCol   : 0
TxPause    : 0
Tx64Byte   : 0
Tx65Byte   : 0
Tx128Byte  : 0
Tx256Byte  : 0
Tx512Byte  : 0
Tx1024Byte : 0
TxByte     : 0
RxDrop     : 0
RxFiltered : 0
RxUni      : 0
RxMulti    : 0
RxBroad    : 0
RxAlignErr : 0
RxCRC      : 0
RxUnderSize: 0
RxFragment : 0
RxOverSize : 0
RxJabber   : 0
RxPause    : 0
Rx64Byte   : 0
Rx65Byte   : 0
Rx128Byte  : 0
Rx256Byte  : 0
Rx512Byte  : 0
Rx1024Byte : 0
RxByte     : 0
RxCtrlDrop : 0
RxIngDrop  : 0
RxARLDrop  : 0

	pvid: 0
	link: port:7 link:down
VLAN 1:
	vid: 0
	ports: 1 6t
VLAN 2:
	vid: 0
	ports: 2 6t
VLAN 3:
	vid: 0
	ports: 3 6t
VLAN 4:
	vid: 0
	ports: 4 6t
VLAN 5:
	vid: 0
	ports: 0 6t
`
		})
		It("returns eight entries", func() {
			Expect(len(ports)).To(Equal(8))
		})
		It("sets link status and speed", func() {
			port0, exists := ports["0"]
			Expect(exists).To(BeTrue())
			Expect(port0.Link).To(Equal("up"))
			Expect(port0.Speed).To(Equal("1000baseT full-duplex"))

			port1, exists := ports["1"]
			Expect(exists).To(BeTrue())
			Expect(port1.Link).To(Equal("up"))
			Expect(port1.Speed).To(Equal("1000baseT full-duplex"))

			port5, exists := ports["5"]
			Expect(exists).To(BeTrue())
			Expect(port5.Link).To(Equal("down"))

			port7, exists := ports["7"]
			Expect(exists).To(BeTrue())
			Expect(port7.Link).To(Equal("down"))
		})
		It("returns mapping from connected mac addresses to port numbers", func() {
			Expect(len(macs)).To(Equal(3))

			Expect(macs["aa:aa:aa:aa:aa:aa"]).To(Equal("0"))
			Expect(macs["bb:bb:bb:bb:bb:bb"]).To(Equal("1"))
			Expect(macs["cc:cc:cc:cc:cc:cc"]).To(Equal("1"))
		})
		Context("with cpu ports", func() {
			BeforeEach(func() {
				cpuPorts = []string{"0", "6"}
			})
			It("returns six entries", func() {
				Expect(len(ports)).To(Equal(6))
			})
			It("removes ports 0 and 6 from the ports map", func() {
				_, exists := ports["0"]
				Expect(exists).To(BeFalse())

				_, exists = ports["6"]
				Expect(exists).To(BeFalse())
			})
			It("removes port 0 from the mac address map", func() {
				Expect(len(macs)).To(Equal(2))

				Expect(macs["bb:bb:bb:bb:bb:bb"]).To(Equal("1"))
				Expect(macs["cc:cc:cc:cc:cc:cc"]).To(Equal("1"))
			})
		})
	})
})
