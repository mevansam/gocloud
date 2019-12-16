package provider_test

import (
	"encoding/json"
	"strings"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/mevansam/gocloud/test/data"
)

var _ = Describe("AWS Provider Tests", func() {

	var (
		err error

		outputBuffer strings.Builder
		awsProvider  provider.CloudProvider
	)

	BeforeEach(func() {
		outputBuffer.Reset()

		awsProvider, err = provider.NewCloudProvider("aws")
		Expect(err).NotTo(HaveOccurred())
		Expect(awsProvider).ToNot(BeNil())
	})

	Context("aws cloud config reference", func() {

		It("retrieves the AWS region information", func() {

			var (
				inputForm        forms.InputForm
				regionInputField *forms.InputField
				acceptedValues   *[]string
			)

			expectedList := [][]string{
				[]string{"ap-east-1", "Asia Pacific (Hong Kong)"},
				[]string{"ap-northeast-1", "Asia Pacific (Tokyo)"},
				[]string{"ap-northeast-2", "Asia Pacific (Seoul)"},
				[]string{"ap-south-1", "Asia Pacific (Mumbai)"},
				[]string{"ap-southeast-1", "Asia Pacific (Singapore)"},
				[]string{"ap-southeast-2", "Asia Pacific (Sydney)"},
				[]string{"ca-central-1", "Canada (Central)"},
				[]string{"eu-central-1", "EU (Frankfurt)"},
				[]string{"eu-north-1", "EU (Stockholm)"},
				[]string{"eu-west-1", "EU (Ireland)"},
				[]string{"eu-west-2", "EU (London)"},
				[]string{"eu-west-3", "EU (Paris)"},
				[]string{"me-south-1", "Middle East (Bahrain)"},
				[]string{"sa-east-1", "South America (Sao Paulo)"},
				[]string{"us-east-1", "US East (N. Virginia)"},
				[]string{"us-east-2", "US East (Ohio)"},
				[]string{"us-west-1", "US West (N. California)"},
				[]string{"us-west-2", "US West (Oregon)"},
			}

			logger.DebugMessage("\nAWS regions retrieved from API call:")

			for i, r := range awsProvider.Regions() {
				logger.DebugMessage("  * %s - %s", r.Name, r.Description)
				Expect(r.Name).To(Equal(expectedList[i][0]))
				Expect(r.Description).To(Equal(expectedList[i][1]))
			}

			inputForm, err = awsProvider.InputForm()
			Expect(err).NotTo(HaveOccurred())
			regionInputField, err = inputForm.GetInputField("region")
			Expect(err).NotTo(HaveOccurred())
			acceptedValues = regionInputField.AcceptedValues()
			Expect(len(*acceptedValues)).To(Equal(len(expectedList)))

			for _, r := range *acceptedValues {
				found := false
				for _, rr := range expectedList {
					if r == rr[0] {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			}
		})
	})

	Context("aws cloud config inputs", func() {

		It("outputs a detailed input data form reference for aws provider config inputs", func() {
			testConfigReferenceOutput(awsProvider, awsInputDataReferenceOutput)
		})

		It("loads configuration values", func() {

			parseConfigDocument(awsProvider, awsConfigDocument, "awsProvider")
			test_data.ValidateAWSConfigDocument(awsProvider)

			// Run some negative tests
			_, err = awsProvider.GetValue("non_existent_key")
			Expect(err).To(HaveOccurred())
		})

		It("saves configuration values", func() {

			var (
				buffer strings.Builder
			)

			parseConfigDocument(awsProvider, awsConfigDocument, "awsProvider")
			_, err = awsProvider.InputForm() // ensure defaults are bound
			writeConfigDocument(awsProvider, "awsProvider", &buffer)

			actual := make(map[string]interface{})
			err = json.Unmarshal([]byte(buffer.String()), &actual)
			Expect(err).NotTo(HaveOccurred())
			logger.TraceMessage("Parsed saved AWS provider config: %# v", actual)

			expected := make(map[string]interface{})
			err = json.Unmarshal([]byte(expectedAWSConfigDocument), &expected)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})
	})

	It("creates a copy of itself", func() {

		var (
			inputForm forms.InputForm

			// value  string
			v1, v2 *string
		)

		parseConfigDocument(awsProvider, awsConfigDocument, "awsProvider")
		copy, err := awsProvider.Copy()
		Expect(err).NotTo(HaveOccurred())

		inputForm, err = awsProvider.InputForm()
		Expect(err).NotTo(HaveOccurred())

		for _, f := range inputForm.InputFields() {

			v1, err = awsProvider.GetValue(f.Name())
			Expect(err).NotTo(HaveOccurred())

			v2, err = copy.GetValue(f.Name())
			Expect(err).NotTo(HaveOccurred())

			Expect(*v2).To(Equal(*v1))
		}

		// Retrieve form again to ensure form is bound to config
		inputForm, err = awsProvider.InputForm()
		Expect(err).NotTo(HaveOccurred())

		err = inputForm.SetFieldValue("access_key", "random value for access_key")
		Expect(err).NotTo(HaveOccurred())

		// Change value in source config
		v1, err = awsProvider.GetValue("access_key")
		Expect(err).NotTo(HaveOccurred())
		Expect(*v1).To(Equal("random value for access_key"))

		// Validate change does not affect copy
		v2, err = copy.GetValue("access_key")
		Expect(err).NotTo(HaveOccurred())
		Expect(*v2).To(Equal("83BFAD5B-FEAC-4019-A645-3858847CB3ED"))
	})
})

const awsInputDataReferenceOutput = `Cloud Provider Configuration
============================

Amazon Web Services Cloud Platform

CONFIGURATION DATA INPUT REFERENCE

* Access Key - The AWS user account's access key id. It will be sourced from the
               environment variable AWS_ACCESS_KEY_ID if not provided.
* Secret Key - The AWS user account's secret key. It will be sourced from the
               environment variable AWS_SECRET_ACCESS_KEY if not provided.
* Region     - The AWS region to create resources in. It will be sourced from
               the environment variable AWS_DEFAULT_REGION if not provided.
* Token      - AWS multi-factor authentication token. It will be sourced from
               the environment variable AWS_SESSION_TOKEN if not provided.`

const awsConfigDocument = `
{
	"cloud": {
		"providers": {
			"awsProvider": ` + test_data.AWSProviderConfig1 + `
		}
	}
}
`

const expectedAWSConfigDocument = `
{
	"cloud": {
		"providers": {
			"awsProvider": ` + test_data.ExpectedAWSProviderConfig1 + `
		}
	}
}			
`
