package provider_test

import (
	"strings"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/term"

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
				acceptedValues   []string
			)

			expectedList := [][]string{
				{"af-south-1", "Africa (Cape Town)"},
				{"ap-east-1", "Asia Pacific (Hong Kong)"},
				{"ap-northeast-1", "Asia Pacific (Tokyo)"},
				{"ap-northeast-2", "Asia Pacific (Seoul)"},
				{"ap-northeast-3", "Asia Pacific (Osaka)"},
				{"ap-south-1", "Asia Pacific (Mumbai)"},
				{"ap-southeast-1", "Asia Pacific (Singapore)"},
				{"ap-southeast-2", "Asia Pacific (Sydney)"},
				{"ap-southeast-3", "Asia Pacific (Jakarta)"},
				{"ca-central-1", "Canada (Central)"},
				{"eu-central-1", "Europe (Frankfurt)"},
				{"eu-north-1", "Europe (Stockholm)"},
				{"eu-south-1", "Europe (Milan)"},
				{"eu-west-1", "Europe (Ireland)"},
				{"eu-west-2", "Europe (London)"},
				{"eu-west-3", "Europe (Paris)"},
				{"me-south-1", "Middle East (Bahrain)"},
				{"sa-east-1", "South America (Sao Paulo)"},
				{"us-east-1", "US East (N. Virginia)"},
				{"us-east-2", "US East (Ohio)"},
				{"us-west-1", "US West (N. California)"},
				{"us-west-2", "US West (Oregon)"},
			}

			logger.DebugMessage("\nAWS regions retrieved from API call:")

			for i, r := range awsProvider.GetRegions() {
				logger.DebugMessage("  * %s - %s", r.Name, r.Description)
				Expect(r.Name).To(Equal(expectedList[i][0]))
				Expect(r.Description).To(Equal(expectedList[i][1]))
			}

			inputForm, err = awsProvider.InputForm()
			Expect(err).NotTo(HaveOccurred())
			regionInputField, err = inputForm.GetInputField("region")
			Expect(err).NotTo(HaveOccurred())
			acceptedValues = regionInputField.AcceptedValues()
			Expect(len(acceptedValues)).To(Equal(len(expectedList)))

			for _, r := range acceptedValues {
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

	Context("aws provider config inputs", func() {

		It("outputs a detailed input data form reference for aws provider config inputs", func() {
			testConfigReferenceOutput(awsProvider, awsInputDataReferenceOutput)
		})

		It("loads configuration values", func() {

			test_data.ParseConfigDocument(awsProvider, awsConfigDocument, "awsProvider")
			test_data.ValidateAWSConfigDocument(awsProvider)

			// Run some negative tests
			_, err = awsProvider.GetValue("non_existent_key")
			Expect(err).To(HaveOccurred())
		})

		It("saves configuration values", func() {
			test_data.ParseConfigDocument(awsProvider, awsConfigDocument, "awsProvider")
			test_data.MarshalConfigDocumentAndValidate(awsProvider, "awsProvider", awsConfigDocument)
		})
	})

	It("creates a copy of itself", func() {
		test_data.ParseConfigDocument(awsProvider, awsConfigDocument, "awsProvider")
		test_data.CopyConfigAndValidate(awsProvider, "access_key", "83BFAD5B-FEAC-4019-A645-3858847CB3ED", "random value for access_key")
	})
})

const awsInputDataReferenceOutput = term.BOLD + `Cloud Provider Configuration
============================` + term.NC + `

Amazon Web Services Cloud Platform

` + term.ITALIC + `CONFIGURATION DATA INPUT REFERENCE` + term.NC + `

* Access Key - The AWS account's access key id. It will be sourced from the
               environment variable AWS_ACCESS_KEY_ID if not provided.
* Secret Key - The AWS account's secret key. It will be sourced from the
               environment variable AWS_SECRET_ACCESS_KEY if not provided.
* Token      - AWS multi-factor authentication token. It will be sourced from
               the environment variable AWS_SESSION_TOKEN if not provided.
* Region     - The AWS region to create resources in. It will be sourced from
               the environment variable AWS_DEFAULT_REGION if not provided.`

const awsConfigDocument = `
{
	"cloud": {
		"awsProvider": ` + test_data.AWSProviderConfig + `
	}
}
`
