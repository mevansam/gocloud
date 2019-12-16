package data

import (
	"github.com/mevansam/gocloud/provider"

	. "github.com/onsi/gomega"
)

// aws provider test data

const AWSProviderConfig = `
{
	"access_key": "83BFAD5B-FEAC-4019-A645-3858847CB3ED",
	"secret_key": "3BA9D494-5D49-4F1A-84CA-70D10A08ACDE",
	"region": "us-east-1",
	"token": "E4B22688-A369-4FB1-B375-732ACED7156F"
}
`

const ExpectedAWSProviderConfig = `
{
	"access_key": "83BFAD5B-FEAC-4019-A645-3858847CB3ED",
	"secret_key": "3BA9D494-5D49-4F1A-84CA-70D10A08ACDE",
	"region": "us-east-1",
	"token": "E4B22688-A369-4FB1-B375-732ACED7156F"
}
`

func ValidateAWSConfigDocument(awsProvider provider.CloudProvider) {

	var (
		err   error
		value *string
	)

	Expect(awsProvider.IsValid()).To(BeTrue())

	value, err = awsProvider.GetValue("access_key")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("83BFAD5B-FEAC-4019-A645-3858847CB3ED"))

	value, err = awsProvider.GetValue("secret_key")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("3BA9D494-5D49-4F1A-84CA-70D10A08ACDE"))

	value, err = awsProvider.GetValue("region")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("us-east-1"))

	value, err = awsProvider.GetValue("token")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("E4B22688-A369-4FB1-B375-732ACED7156F"))
}

// google provider test data

const GoogleProviderConfig = `
{
	"authentication": {
		"credentials": "/home/username/gcp-service-account.json",
		"access_token": "0640E5A6-8346-4F99-9ED7-7E384CCD0EAA"
	},
	"project": "my-google-project",
	"region": "europe-west1",
	"zone": "europe-west1-b"
}
`

func ValidateGoogleConfigDocument(googleProvider provider.CloudProvider) {

	var (
		err   error
		value *string
	)

	Expect(googleProvider.IsValid()).To(BeTrue())

	value, err = googleProvider.GetValue("credentials")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("/home/username/gcp-service-account.json"))

	value, err = googleProvider.GetValue("access_token")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("0640E5A6-8346-4F99-9ED7-7E384CCD0EAA"))

	value, err = googleProvider.GetValue("project")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("my-google-project"))

	value, err = googleProvider.GetValue("region")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("europe-west1"))

	value, err = googleProvider.GetValue("zone")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("europe-west1-b"))
}

// azure provider test data

const AzureProviderConfig = `
{
	"environment": "government",
	"subscription_id": "33EA4208-F718-4206-94E4-8A06E041858E",
	"client_id": "BC3974F4-02C2-4762-8561-8CD466450914",
	"client_secret": "98D1264D-D4D0-4BC4-8038-F671F9DAE3D1",
	"tenant_id": "185C4F21-8C20-4694-B450-A9ED71321C6E",
	"default_resource_group": "cb_default_b602e51d27ad4c338092464590d29aef",
	"default_location": "westus"
}
`

func ValidateAzureConfigDocument(azureProvider provider.CloudProvider) {

	var (
		err   error
		value *string
	)

	Expect(azureProvider.IsValid()).To(BeTrue())

	value, err = azureProvider.GetValue("environment")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("government"))

	value, err = azureProvider.GetValue("subscription_id")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("33EA4208-F718-4206-94E4-8A06E041858E"))

	value, err = azureProvider.GetValue("client_id")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("BC3974F4-02C2-4762-8561-8CD466450914"))

	value, err = azureProvider.GetValue("client_secret")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("98D1264D-D4D0-4BC4-8038-F671F9DAE3D1"))

	value, err = azureProvider.GetValue("tenant_id")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("185C4F21-8C20-4694-B450-A9ED71321C6E"))

	value, err = azureProvider.GetValue("default_resource_group")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("cb_default_b602e51d27ad4c338092464590d29aef"))

	value, err = azureProvider.GetValue("default_location")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).NotTo(BeNil())
	Expect(*value).To(Equal("westus"))
}
