package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/mevansam/gocloud/test/data"
	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

var _ = Describe("Azure Provider Tests", func() {

	var (
		err error

		outputBuffer  strings.Builder
		azureProvider provider.CloudProvider
	)

	BeforeEach(func() {
		outputBuffer.Reset()

		azureProvider, err = provider.NewCloudProvider("azure")
		Expect(err).NotTo(HaveOccurred())
		Expect(azureProvider).ToNot(BeNil())
	})

	Context("azure cloud reference", func() {

		var (
			azurePublicLocations      map[string]string
			azureDefaultResourceGroup string
		)

		BeforeEach(func() {

			var (
				inputForm forms.InputForm
			)

			azurePublicLocations = test_helpers.AzureLocations("AzurePublicCloud")
			test_helpers.InitializeAzureProvider(azureProvider)

			// reset resource group in the case ARM_NO_CLEANUP is specified
			// as that is there to optimize the cloud tests only
			inputForm, err = azureProvider.InputForm()
			Expect(err).NotTo(HaveOccurred())

			azureDefaultResourceGroup = fmt.Sprintf("cb_default_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
			err = inputForm.SetFieldValue("default_resource_group", azureDefaultResourceGroup)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			test_helpers.DeleteAzureResourceGroup(azureDefaultResourceGroup)
		})

		It("creates the default resource group and storage account", func() {

			// Connect provider to Azure service
			err = azureProvider.Connect()
			Expect(err).NotTo(HaveOccurred())

			defaultResourceGroup, err := azureProvider.GetValue("default_resource_group")
			Expect(err).NotTo(HaveOccurred())

			// Validate default resource group was created
			client := test_helpers.AzureGroupsClient("AzurePublicCloud")
			group, err := client.Get(context.Background(), *defaultResourceGroup)
			Expect(err).NotTo(HaveOccurred())
			Expect(*group.Name).To(Equal(*defaultResourceGroup))

			storageAccountName := provider.GetAzureStorageAccountName(azureProvider)
			Expect(
				test_helpers.AzureStorageAccountExists(storageAccountName, *defaultResourceGroup),
			).To(BeFalse())

			storage, err := azureProvider.GetStorage()
			Expect(err).NotTo(HaveOccurred())
			Expect(storage).ToNot(BeNil())

			storageAccountName = provider.GetAzureStorageAccountName(azureProvider)
			Expect(
				test_helpers.AzureStorageAccountExists(storageAccountName, *defaultResourceGroup),
			).To(BeTrue())
		})

		It("retrieves the Azure region information and validates against static list", func() {

			regionInfoList := azureProvider.Regions()
			Expect(len(regionInfoList)).To(Equal(len(azurePublicLocations)))

			for _, r := range regionInfoList {
				Expect(r.Description).To(Equal(azurePublicLocations[r.Name]))
			}
		})

		It("retrieves the Azure region information from an authenticated provider", func() {

			// Connect provider to Azure service
			err = azureProvider.Connect()
			Expect(err).NotTo(HaveOccurred())

			regionInfoList := azureProvider.Regions()
			Expect(len(regionInfoList)).To(Equal(len(azurePublicLocations)))

			for _, r := range regionInfoList {
				Expect(r.Description).To(Equal(azurePublicLocations[r.Name]))
			}
		})
	})

	Context("azure cloud config", func() {

		It("outputs a detailed input data form reference for azure provider config inputs", func() {
			testConfigReferenceOutput(azureProvider, azureInputDataReferenceOutput)
		})

		It("loads a configuration values", func() {

			parseConfigDocument(azureProvider, azureConfigDocument, "azureProvider")
			test_data.ValidateAzureConfigDocument(azureProvider)

			// Run some negative tests
			_, err = azureProvider.GetValue("non_existent_key")
			Expect(err).To(HaveOccurred())
		})

		It("saves a configuration values", func() {

			var (
				buffer strings.Builder
			)

			parseConfigDocument(azureProvider, azureConfigDocument, "azureProvider")
			writeConfigDocument(azureProvider, "azureProvider", &buffer)

			actual := make(map[string]interface{})
			err = json.Unmarshal([]byte(buffer.String()), &actual)
			Expect(err).NotTo(HaveOccurred())
			logger.TraceMessage("Parsed saved Azure provider config: %# v", actual)

			expected := make(map[string]interface{})
			err = json.Unmarshal([]byte(azureConfigDocument), &expected)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})

		It("creates a copy of itself", func() {

			var (
				inputForm forms.InputForm

				// value  string
				v1, v2 *string
			)

			parseConfigDocument(azureProvider, azureConfigDocument, "azureProvider")
			copy, err := azureProvider.Copy()
			Expect(err).NotTo(HaveOccurred())

			inputForm, err = azureProvider.InputForm()
			Expect(err).NotTo(HaveOccurred())

			for _, f := range inputForm.InputFields() {

				v1, err = azureProvider.GetValue(f.Name())
				Expect(err).NotTo(HaveOccurred())

				v2, err = copy.GetValue(f.Name())
				Expect(err).NotTo(HaveOccurred())

				Expect(*v2).To(Equal(*v1))
			}

			// Retrieve form again to ensure form is bound to config
			inputForm, err = azureProvider.InputForm()
			Expect(err).NotTo(HaveOccurred())

			err = inputForm.SetFieldValue("client_id", "random value for client_id")
			Expect(err).NotTo(HaveOccurred())

			// Change value in source config
			v1, err = azureProvider.GetValue("client_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(*v1).To(Equal("random value for client_id"))

			// Validate change does not affect copy
			v2, err = copy.GetValue("client_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(*v2).To(Equal("BC3974F4-02C2-4762-8561-8CD466450914"))
		})
	})
})

const azureInputDataReferenceOutput = `Cloud Provider Configuration
============================

Microsoft Azure Cloud Computing Platform

CONFIGURATION DATA INPUT REFERENCE

* Environment                - The Azure environment. It will be sourced from
                               the environment variable ARM_ENVIRONMENT if not
                               provided.
* Subscription ID            - The Azure subscription ID. It will be sourced
                               from the environment variable ARM_SUBSCRIPTION_ID
                               if not provided.
* Client ID                  - The Client ID or Application ID of the Azure
                               Service Principal. It will be sourced from the
                               environment variable ARM_CLIENT_ID if not
                               provided.
* Client Secret              - The Client Secret or Password of the Azure
                               Service Principal. It will be sourced from the
                               environment variable ARM_CLIENT_SECRET if not
                               provided.
* Tenant ID                  - The Tenant ID from the Azure Service Principal.
                               It will be sourced from the environment variable
                               ARM_TENANT_ID if not provided.
* Default Resource Group     - Resource group where common resources will be
                               created.
* Default Location or Region - The location of the default resource group.`

const azureConfigDocument = `
{
	"cloud": {
		"providers": {
			"azureProvider": ` + test_data.AzureProviderConfig + `
		}
	}
}
`

// Azure SDK Helper functions
