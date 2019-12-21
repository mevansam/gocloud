package backend_test

import (
	"strings"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"

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

	Context("initialization", func() {

		It("the cloud provider must match the backend cloud requirement", func() {

			awsProvider, err := provider.NewCloudProvider("aws")
			Expect(err).NotTo(HaveOccurred())
			Expect(awsProvider).ToNot(BeNil())
		})

		It("can be inititialized using the correct cloud provider", func() {

			var (
				inputForm forms.InputForm
				value     *string
			)
			azureProvider, err := provider.NewCloudProvider("azure")
			Expect(err).NotTo(HaveOccurred())
			Expect(azureProvider).ToNot(BeNil())
			test_data.ParseConfigDocument(azureProvider, azureConfigDocument, "azureProvider")

			err = azurermBackend.Configure(azureProvider, "mybackend", "mystatekey")
			Expect(err).NotTo(HaveOccurred())

			inputForm, err = azurermBackend.InputForm()
			Expect(err).NotTo(HaveOccurred())

			value, err = inputForm.GetFieldValue("resource_group_name")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("cb_default_b602e51d27ad4c338092464590d29aef"))

			value, err = inputForm.GetFieldValue("storage_account_name")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("cbdefaultb602e51d27ad4c3"))

			value, err = inputForm.GetFieldValue("container_name")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("mybackend-westus"))

			value, err = inputForm.GetFieldValue("key")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("mystatekey"))
		})
	})

	Context("copying", func() {

		It("can creates a copy of itself", func() {
			test_data.ParseConfigDocument(azurermBackend, azurermConfigDocument, "azurermBackend")
			test_data.CopyConfigAndValidate(azurermBackend, "container_name", "mystatecontainer", "newcontainer")
		})
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

const azureConfigDocument = `
{
	"cloud": {
		"azureProvider": ` + test_data.AzureProviderConfig + `
	}
}
`
