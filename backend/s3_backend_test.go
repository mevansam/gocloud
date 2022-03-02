package backend_test

import (
	"strings"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/term"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/mevansam/gocloud/test/data"
)

var _ = Describe("S3 Backend Tests", func() {

	var (
		err error

		outputBuffer strings.Builder
		s3Backend    backend.CloudBackend
	)

	BeforeEach(func() {
		outputBuffer.Reset()

		s3Backend, err = backend.NewCloudBackend("s3")
		Expect(err).NotTo(HaveOccurred())
		Expect(s3Backend).ToNot(BeNil())
	})

	Context("s3 backend config inputs", func() {

		It("outputs a detailed input data form reference for aws provider config inputs", func() {
			testConfigReferenceOutput(s3Backend, s3InputDataReferenceOutput)
		})

		It("loads configuration values", func() {

			test_data.ParseConfigDocument(s3Backend, s3ConfigDocument, "s3Backend")
			test_data.ValidateS3ConfigDocument(s3Backend)

			// Run a negative test
			_, err = s3Backend.GetValue("non_existent_key")
			Expect(err).To(HaveOccurred())
		})

		It("saves configuration values", func() {
			test_data.ParseConfigDocument(s3Backend, s3ConfigDocument, "s3Backend")
			test_data.MarshalConfigDocumentAndValidate(s3Backend, "s3Backend", s3ConfigDocument)
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
			awsProvider, err := provider.NewCloudProvider("aws")
			Expect(err).NotTo(HaveOccurred())
			Expect(awsProvider).ToNot(BeNil())
			test_data.ParseConfigDocument(awsProvider, awsConfigDocument, "awsProvider")

			err = s3Backend.Configure(awsProvider, "mybackend", "mystatekey")
			Expect(err).NotTo(HaveOccurred())

			inputForm, err = s3Backend.InputForm()
			Expect(err).NotTo(HaveOccurred())

			value, err = inputForm.GetFieldValue("bucket")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(MatchRegexp(`^mybackend-us-east-1-[0-9a-f]{32}$`))

			value, err = inputForm.GetFieldValue("key")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("mystatekey"))
		})
	})

	Context("copying", func() {

		It("can creates a copy of itself", func() {
			test_data.ParseConfigDocument(s3Backend, s3ConfigDocument, "s3Backend")
			test_data.CopyConfigAndValidate(s3Backend, "bucket", "mystatebucket", "newbucket")
		})
	})
})

const s3InputDataReferenceOutput = term.BOLD + `Cloud Backend Configuration
===========================` + term.NC + `

Amazon Web Services S3 Storage Backend

` + term.ITALIC + `CONFIGURATION DATA INPUT REFERENCE` + term.NC + `

* Bucket - The S3 bucket to store state in.
* Key    - The key with which to identify the state object in the bucket.`

const s3ConfigDocument = `
{
	"cloud": {
		"s3Backend": ` + test_data.S3BackendConfig + `
	}
}
`

const awsConfigDocument = `
{
	"cloud": {
		"awsProvider": ` + test_data.AWSProviderConfig + `
	}
}
`
