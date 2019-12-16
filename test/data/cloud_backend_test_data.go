package data

import (
	"github.com/mevansam/gocloud/backend"

	. "github.com/onsi/gomega"
)

// s3 provider test data

const S3BackendConfig = `
{
	"bucket": "mystatebucket",
	"key": "mystatekey"
}
`

const ExpectedS3BackendConfig = `
{
	"bucket": "mystatebucket",
	"key": "mystatekey"
}
`

func ValidateS3ConfigDocument(s3Backend backend.CloudBackend) {

	var (
		err   error
		value *string
	)

	Expect(s3Backend.IsValid()).To(BeTrue())

	value, err = s3Backend.GetValue("bucket")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("mystatebucket"))

	value, err = s3Backend.GetValue("key")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("mystatekey"))
}

// azurerm provider test data

const AzureRMBackendConfig = `
{
	"resource_group_name": "myresourcegroup",
	"storage_account_name": "mystorageaccount",
	"container_name": "mystatecontainer",
	"key": "mystatekey"
}
`

const ExpectedAzureRMBackendConfig = `
{
	"resource_group_name": "myresourcegroup",
	"storage_account_name": "mystorageaccount",
	"container_name": "mystatecontainer",
	"key": "mystatekey"
}
`

func ValidateAzureRMConfigDocument(azurermBackend backend.CloudBackend) {

	var (
		err   error
		value *string
	)

	Expect(azurermBackend.IsValid()).To(BeTrue())

	value, err = azurermBackend.GetValue("resource_group_name")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("myresourcegroup"))

	value, err = azurermBackend.GetValue("storage_account_name")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("mystorageaccount"))

	value, err = azurermBackend.GetValue("container_name")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("mystatecontainer"))

	value, err = azurermBackend.GetValue("key")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("mystatekey"))
}

// gcs provider test data

const GCSBackendConfig = `
{
	"bucket": "mystatebucket",
	"prefix": "mystateprefix"
}
`

const ExpectedGCSBackendConfig = `
{
	"bucket": "mystatebucket",
	"prefix": "mystateprefix"
}
`

func ValidateGCSConfigDocument(gcsBackend backend.CloudBackend) {

	var (
		err   error
		value *string
	)

	Expect(gcsBackend.IsValid()).To(BeTrue())

	value, err = gcsBackend.GetValue("bucket")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("mystatebucket"))

	value, err = gcsBackend.GetValue("prefix")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("mystateprefix"))
}
