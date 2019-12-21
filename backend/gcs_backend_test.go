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

var _ = Describe("Google Cloud Storage Backend Tests", func() {

	var (
		err error

		outputBuffer strings.Builder
		gcsBackend   backend.CloudBackend
	)

	BeforeEach(func() {
		outputBuffer.Reset()

		gcsBackend, err = backend.NewCloudBackend("gcs")
		Expect(err).NotTo(HaveOccurred())
		Expect(gcsBackend).ToNot(BeNil())
	})

	Context("gcs backend config inputs", func() {

		It("outputs a detailed input data form reference for aws provider config inputs", func() {
			testConfigReferenceOutput(gcsBackend, gcsInputDataReferenceOutput)
		})

		It("loads configuration values", func() {

			test_data.ParseConfigDocument(gcsBackend, gcsConfigDocument, "gcsBackend")
			test_data.ValidateGCSConfigDocument(gcsBackend)

			// Run some negative tests
			_, err = gcsBackend.GetValue("non_existent_key")
			Expect(err).To(HaveOccurred())
		})

		It("saves configuration values", func() {
			test_data.ParseConfigDocument(gcsBackend, gcsConfigDocument, "gcsBackend")
			test_data.MarshalConfigDocumentAndValidate(gcsBackend, "gcsBackend", gcsConfigDocument)
		})
	})

	Context("initialization", func() {

		It("the cloud provider must match the backend cloud requirement", func() {

			azureProvider, err := provider.NewCloudProvider("azure")
			Expect(err).NotTo(HaveOccurred())
			Expect(azureProvider).ToNot(BeNil())
		})

		It("can be inititialized using the correct cloud provider", func() {

			var (
				inputForm forms.InputForm
				value     *string
			)
			googleProvider, err := provider.NewCloudProvider("google")
			Expect(err).NotTo(HaveOccurred())
			Expect(googleProvider).ToNot(BeNil())
			test_data.ParseConfigDocument(googleProvider, googleConfigDocument, "googleProvider")

			err = gcsBackend.Configure(googleProvider, "mybackend", "mystatekey")
			Expect(err).NotTo(HaveOccurred())

			inputForm, err = gcsBackend.InputForm()
			Expect(err).NotTo(HaveOccurred())

			value, err = inputForm.GetFieldValue("bucket")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("mybackend-europe-west1"))

			value, err = inputForm.GetFieldValue("prefix")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("mystatekey"))
		})
	})

	Context("copying", func() {

		It("can creates a copy of itself", func() {
			test_data.ParseConfigDocument(gcsBackend, gcsConfigDocument, "gcsBackend")
			test_data.CopyConfigAndValidate(gcsBackend, "bucket", "mystatebucket", "newbucket")
		})
	})
})

const gcsInputDataReferenceOutput = `Cloud Backend Configuration
===========================

Google Cloud Storage Backend

CONFIGURATION DATA INPUT REFERENCE

* Bucket - The GCS bucket to store state in.
* Prefix - The prefix to use in the name of the state object.`

const gcsConfigDocument = `
{
	"cloud": {
		"gcsBackend": ` + test_data.GCSBackendConfig + `
	}
}
`

const googleConfigDocument = `
{
	"cloud": {
		"googleProvider": ` + test_data.GoogleProviderConfig + `
	}
}
`
