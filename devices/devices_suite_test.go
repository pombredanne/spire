package devices_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/service"
)

// TestHandlers ...
func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spire Devices Suite")
}

var mockLiberator *httptest.Server

var _ = BeforeSuite(func() {
	mockLiberator = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(`
			{
				"data": {
					"current_system_image": {
						"product":"archer-c7",
						"variant":"lingrush",
						"vendor":"tplink",
						"version":44
					}
				}
			}`))
	}))

	service.Config.LiberatorBaseURL = mockLiberator.URL
	service.Config.Environment = "test"
})

var _ = AfterSuite(func() {
	mockLiberator.Close()
})
