package backend_test

import (
	"strings"

	"github.com/mevansam/gocloud/backend"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/mevansam/gocloud/test/data"
)

var _ = Describe("Azure Backend Tests", func() {

	var (
		err error

		outputBuffer   strings.Builder
		azurermBackend backend.CloudBackend
	)

	BeforeEach(func() {
		outputBuffer.Reset()

		azurermBackend, err = backend.NewCloudBackend("azurerm")
		Expect(err).NotTo(HaveOccurred())
		Expect(azurermBackend).ToNot(BeNil())
	})

	Context("azure backend config inputs", func() {

		It("outputs a detailed input data form reference for aws provider config inputs", func() {
			testConfigReferenceOutput(azurermBackend, azurermInputDataReferenceOutput)
		})

		It("loads configuration values", func() {

			test_data.ParseConfigDocument(azurermBackend, azurermConfigDocument, "azurermBackend")
			test_data.ValidateAzureRMConfigDocument(azurermBackend)

			// Run some negative tests
			_, err = azurermBackend.GetValue("non_existent_key")
			Expect(err).To(HaveOccurred())
		})

		It("saves configuration values", func() {
			test_data.ParseConfigDocument(azurermBackend, azurermConfigDocument, "azurermBackend")
			test_data.MarshalConfigDocumentAndValidate(azurermBackend, "azurermBackend", azurermConfigDocument)
		})
	})

	It("creates a copy of itself", func() {
		test_data.ParseConfigDocument(azurermBackend, azurermConfigDocument, "azurermBackend")
		test_data.CopyConfigAndValidate(azurermBackend, "container_name", "mystatecontainer", "newcontainer")
	})
})

const azurermInputDataReferenceOutput = `Cloud Backend Configuration
===========================

Azure Resource Manager Storage Backend

CONFIGURATION DATA INPUT REFERENCE

* Resource Group Name  - The Azure resource group name where storage resources
                         will be created.
* Storage Account Name - The name of the storage account to use for the state
                         container.
* Container Name       - The name of the storage container where state will be
                         saved.
* Key                  - The key with which to identify the state blob in the
                         container.`

const azurermConfigDocument = `
{
	"cloud": {
		"azurermBackend": ` + test_data.AzureRMBackendConfig + `
	}
}
`
