package backend_test

import (
	"strings"

	"github.com/mevansam/gocloud/backend"

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
			test_data.MarshalConfigDocumentAndValidate(s3Backend, "backend", "s3Backend", s3ConfigDocument)
		})
	})

	It("creates a copy of itself", func() {
		test_data.ParseConfigDocument(s3Backend, s3ConfigDocument, "s3Backend")
		test_data.CopyConfigAndValidate(s3Backend, "bucket", "mystatebucket", "newbucket")
	})
})

const s3InputDataReferenceOutput = `Cloud Backend Configuration
===========================

Amazon Web Services S3 Storage Backend

CONFIGURATION DATA INPUT REFERENCE

* Bucket - The S3 bucket to store state in.
* Key    - The key with which to identify the state object in the bucket.`

const s3ConfigDocument = `
{
	"cloud": {
		"backend": {
			"s3Backend": ` + test_data.S3BackendConfig + `
		}
	}
}
`
