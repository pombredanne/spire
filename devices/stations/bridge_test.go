package stations_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/devices/stations"
)

var _ = Describe("Bridge Parser", func() {

	var res map[string]stations.BridgeInfo
	var err error
	var bridgeInput string

	JustBeforeEach(func() {
		res, err = stations.ParseBridgeMACs(bridgeInput)
	})
	Context("invalid message", func() {
		BeforeEach(func() {
			bridgeInput = "read of forward table failed: No such device\n"
		})
		It("returns an empty map of bridge info objects", func() {
			Expect(err).To(BeNil())
			Expect(len(res)).To(BeZero())
		})
	})
	Context("valid message", func() {
		BeforeEach(func() {
			bridgeInput = "port no\tmac addr\t\tis local?\tageing timer\n  4\tff:ff:ff:ff:ff:01\tno\t\t   0.02\n  4\tff:ff:ff:ff:ff:02\tno\t\t   1.51\n  4\tff:ff:ff:ff:ff:03\tno\t\t   4.19\n  4\tff:ff:ff:ff:ff:04\tyes\t\t   0.00\n"
		})
		It("sets 'age'", func() {
			Expect(err).To(BeNil())
			Expect(res["ff:ff:ff:ff:ff:01"].Age).To(Equal(0.02))
			Expect(res["ff:ff:ff:ff:ff:02"].Age).To(Equal(1.51))
			Expect(res["ff:ff:ff:ff:ff:03"].Age).To(Equal(4.19))
			Expect(res["ff:ff:ff:ff:ff:04"].Age).To(Equal(0.00))
		})
		It("sets 'local'", func() {
			Expect(err).To(BeNil())
			Expect(res["ff:ff:ff:ff:ff:01"].Local).To(BeFalse())
			Expect(res["ff:ff:ff:ff:ff:02"].Local).To(BeFalse())
			Expect(res["ff:ff:ff:ff:ff:03"].Local).To(BeFalse())
			Expect(res["ff:ff:ff:ff:ff:04"].Local).To(BeTrue())
		})
	})
})
