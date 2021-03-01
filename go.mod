module github.com/mevansam/gocloud

go 1.16

replace github.com/mevansam/goutils => ../goutils

replace github.com/mevansam/goforms => ../goforms

require (
	cloud.google.com/go/storage v1.4.0
	github.com/Azure/azure-sdk-for-go v37.2.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest v0.9.3
	github.com/Azure/go-autorest/autorest/adal v0.8.1
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/aws/aws-sdk-go v1.27.0
	github.com/google/uuid v1.1.1
	github.com/mevansam/goforms v0.0.0-00010101000000-000000000000
	github.com/mevansam/goutils v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	google.golang.org/api v0.15.0
)
