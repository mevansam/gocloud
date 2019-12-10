package cloud_test

import (
	"github.com/mevansam/goutils/logger"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

var _ = Describe("AWS Compute Tests", func() {

	var (
		err error

		awsProvider provider.CloudProvider
		awsCompute  cloud.Compute

		testInstances map[string]string
	)

	BeforeEach(func() {

		awsProvider, err = provider.NewCloudProvider("aws")
		Expect(err).NotTo(HaveOccurred())
		Expect(awsProvider).ToNot(BeNil())

		test_helpers.InitializeAWSProvider(awsProvider)

		err = awsProvider.Connect()
		Expect(err).NotTo(HaveOccurred())

		awsCompute, err = awsProvider.GetCompute()
		Expect(err).NotTo(HaveOccurred())
		Expect(awsCompute).ToNot(BeNil())

		// ensure 2 test VMs have been created for these tests
		testInstances = test_helpers.AWSDeployTestInstances("test", 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(testInstances)).To(Equal(2))

		logger.TraceMessage("Test VMs: %# v", testInstances)

		awsCompute.SetProperties(cloud.AWSComputeProperties{
			FilterTags: map[string]string{
				"Role": "Cloudbuilder-Test",
			},
		})
	})

	Context("Compute resources", func() {

		It("retrieves a list of compute instances", func() {

			instances, err := awsCompute.ListInstances()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(instances)).To(Equal(len(testInstances)))

			for _, instance := range instances {
				ipAddress, exists := testInstances[instance.Name()]
				Expect(exists).To(BeTrue())
				Expect(instance.PublicIP()).To(Equal(ipAddress))
			}
		})

		It("retrieves a compute instance", func() {

			instance, err := awsCompute.GetInstance("test-X")
			Expect(err).To(HaveOccurred())

			instance, err = awsCompute.GetInstance("test-0")
			Expect(err).NotTo(HaveOccurred())
			Expect(instance).ToNot(BeNil())
		})
	})

	Context("Compute instance", func() {

		var (
			instance0 cloud.ComputeInstance
			instance1 cloud.ComputeInstance
		)

		BeforeEach(func() {
			instance0, err = awsCompute.GetInstance("test-0")
			Expect(err).NotTo(HaveOccurred())
			Expect(instance0).ToNot(BeNil())

			instance1, err = awsCompute.GetInstance("test-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(instance1).ToNot(BeNil())
		})

		It("stops and starts a compute instance", func() {

			state, err := instance0.State()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cloud.StateRunning))

			err = instance0.Stop()
			Expect(err).NotTo(HaveOccurred())

			state, err = instance0.State()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cloud.StateStopped))

			awsState := test_helpers.AWSInstanceState(instance0.ID())
			Expect(awsState).To(Equal("stopped"))

			err = instance0.Start()
			Expect(err).NotTo(HaveOccurred())

			state, err = instance0.State()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cloud.StateRunning))
		})

		It("restarts a compute instance", func() {

			state, err := instance1.State()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cloud.StateRunning))

			err = instance0.Restart()
			Expect(err).NotTo(HaveOccurred())

			state, err = instance1.State()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cloud.StateRunning))
		})
	})
})
