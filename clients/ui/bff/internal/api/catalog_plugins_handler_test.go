package api

import (
	"net/http"

	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/model-registry/ui/bff/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestGetAllCatalogPluginsHandler", func() {
	Context("testing Catalog Plugins Handler", Ordered, func() {

		It("should retrieve all catalog plugins", func() {
			By("fetching all catalog plugins")
			data := mocks.GetCatalogPluginListMock()
			requestIdentity := kubernetes.RequestIdentity{
				UserID: "user@example.com",
			}

			expected := CatalogPluginListEnvelope{Data: &data}
			actual, rs, err := setupApiTest[CatalogPluginListEnvelope](http.MethodGet, "/api/v1/model_catalog/plugins?namespace=kubeflow", nil, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			By("should match the expected catalog plugins")
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(actual.Data.Count).To(Equal(expected.Data.Count))
			Expect(len(actual.Data.Plugins)).To(Equal(len(expected.Data.Plugins)))
			Expect(actual.Data.Plugins[0].Name).To(Equal(expected.Data.Plugins[0].Name))
			Expect(actual.Data.Plugins[0].BasePath).To(Equal(expected.Data.Plugins[0].BasePath))
			Expect(actual.Data.Plugins[0].Healthy).To(Equal(expected.Data.Plugins[0].Healthy))
			Expect(actual.Data.Plugins[1].Name).To(Equal(expected.Data.Plugins[1].Name))
			Expect(actual.Data.Plugins[1].BasePath).To(Equal(expected.Data.Plugins[1].BasePath))
		})

	})
})
