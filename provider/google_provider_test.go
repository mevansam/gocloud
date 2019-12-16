package provider_test

import (
	"strings"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/mevansam/gocloud/test/data"
	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

var _ = Describe("Google Provider Tests", func() {

	var (
		err error

		outputBuffer   strings.Builder
		googleProvider provider.CloudProvider
	)

	BeforeEach(func() {
		outputBuffer.Reset()

		googleProvider, err = provider.NewCloudProvider("google")
		Expect(err).NotTo(HaveOccurred())
		Expect(googleProvider).ToNot(BeNil())
	})

	Context("google cloud config reference", func() {

		var (
			googleRegions []string
		)

		BeforeEach(func() {
			test_helpers.InitializeGoogleProvider(googleProvider)
			googleRegions = test_helpers.GoogleRegions()
		})

		It("retrieves the Google region information", func() {

			var (
				inputForm        forms.InputForm
				regionInputField *forms.InputField
				acceptedValues   *[]string
			)

			regionList := googleProvider.Regions()
			for i, r := range regionList {
				Expect(r.Name).To(Equal(googleRegions[i]))
			}

			inputForm, err = googleProvider.InputForm()
			Expect(err).NotTo(HaveOccurred())
			regionInputField, err = inputForm.GetInputField("region")
			Expect(err).NotTo(HaveOccurred())
			acceptedValues = regionInputField.AcceptedValues()
			Expect(len(*acceptedValues)).To(Equal(len(regionList)))

			for _, r := range *acceptedValues {
				found := false
				for _, rr := range googleRegions {
					if r == rr {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			}
		})

		It("retrieves the Google region information from an authenticated provider", func() {

			// Connect provider to Azure service
			err = googleProvider.Connect()
			Expect(err).NotTo(HaveOccurred())

			regionInfoList := googleProvider.Regions()
			Expect(len(regionInfoList)).To(Equal(len(googleRegions)))

			for _, r := range regionInfoList {
				found := false
				for _, rr := range googleRegions {
					if r.Name == rr {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			}
		})
	})

	Context("google provider config inputs", func() {

		It("outputs a detailed input data form reference for google provider config inputs", func() {
			testConfigReferenceOutput(googleProvider, googleInputDataReferenceOutput)
		})

		It("loads a configuration values", func() {

			test_data.ParseConfigDocument(googleProvider, googleConfigDocument, "googleProvider")
			test_data.ValidateGoogleConfigDocument(googleProvider)

			// Run some negative tests
			_, err = googleProvider.GetValue("non_existent_key")
			Expect(err).To(HaveOccurred())
		})

		It("saves a configuration values", func() {
			test_data.ParseConfigDocument(googleProvider, googleConfigDocument, "googleProvider")
			test_data.MarshalConfigDocumentAndValidate(googleProvider, "providers", "googleProvider", googleConfigDocument)
		})

		It("creates a copy of itself", func() {
			test_data.ParseConfigDocument(googleProvider, googleConfigDocument, "googleProvider")
			test_data.CopyConfigAndValidate(googleProvider, "access_token", "0640E5A6-8346-4F99-9ED7-7E384CCD0EAA", "random value for access_token")
		})
	})
})

const googleInputDataReferenceOutput = `Cloud Provider Configuration
============================

Google Cloud Platform

CONFIGURATION DATA INPUT REFERENCE

* Provide one of the following for:

  Google Cloud authentication credentials

  * Credentials - The contents of a service account key file in JSON format. It
                  will be sourced from the environment variables
                  GOOGLE_CREDENTIALS, GOOGLE_CLOUD_KEYFILE_JSON,
                  GCLOUD_KEYFILE_JSON if not provided.

  OR

  * Access Token - A temporary OAuth 2.0 access token obtained from the Google
                   Authorization server. It will be sourced from the environment
                   variable GOOGLE_OAUTH_ACCESS_TOKEN if not provided.

* Project - The Google Cloud Platform project to manage resources in. It will be
            sourced from the environment variables GOOGLE_PROJECT,
            GOOGLE_CLOUD_PROJECT, GCLOUD_PROJECT, CLOUDSDK_CORE_PROJECT if not
            provided.
* Region  - The default region to manage resources in. It will be sourced from
            the environment variables GOOGLE_REGION, GCLOUD_REGION,
            CLOUDSDK_COMPUTE_REGION if not provided.
* Zone    - The default zone to manage resources in. It will be sourced from the
            environment variables GOOGLE_ZONE, GCLOUD_ZONE,
            CLOUDSDK_COMPUTE_ZONE if not provided.`

const googleConfigDocument = `
{
	"cloud": {
		"providers": {
			"googleProvider": ` + test_data.GoogleProviderConfig + `
		}
	}
}
`
