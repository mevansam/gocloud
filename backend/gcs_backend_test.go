package backend_test

import (
	"strings"

	"github.com/mevansam/gocloud/backend"

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
			test_data.MarshalConfigDocumentAndValidate(gcsBackend, "backend", "gcsBackend", gcsConfigDocument)
		})
	})

	It("creates a copy of itself", func() {
		test_data.ParseConfigDocument(gcsBackend, gcsConfigDocument, "gcsBackend")
		test_data.CopyConfigAndValidate(gcsBackend, "bucket", "mystatebucket", "newbucket")
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
		"backend": {
			"gcsBackend": ` + test_data.GCSBackendConfig + `
		}
	}
}
`
