package ports_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ports"
)

var _ = Describe("Ports", func() {

	When("port mapping is enabled", func() {
		BeforeEach(func() {
			ports.Init(false)
		})

		It("should translate privileged port", func() {
			expectPortTranslated(80)
			expectPortTranslated(443)
			expectPortTranslated(1023)
		})
		It("should NOT translate unprivileged port", func() {
			expectPortNotTranslated(8080)
			expectPortNotTranslated(8443)
			expectPortNotTranslated(1024)
		})
	})

	When("port mapping is disabled", func() {
		BeforeEach(func() {
			ports.Init(true)
		})

		It("should NOT translate privileged port", func() {
			expectPortNotTranslated(80)
			expectPortNotTranslated(443)
			expectPortNotTranslated(1023)
		})
		It("should NOT translate unprivileged port", func() {
			expectPortNotTranslated(8080)
			expectPortNotTranslated(8443)
			expectPortNotTranslated(1024)
		})
	})

})

func expectPortTranslated(port uint16) {
	Expect(ports.TranslatePort(port)).To(Equal(port + ports.PortOffset))
}

func expectPortNotTranslated(port uint16) {
	Expect(ports.TranslatePort(port)).To(Equal(port))
}
