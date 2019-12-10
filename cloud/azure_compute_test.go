package cloud_test

import (
	"github.com/mevansam/goutils/logger"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

var _ = Describe("Azure Compute Tests", func() {

	var (
		err error

		azureProvider provider.CloudProvider
		azureCompute  cloud.Compute

		testInstances map[string]string
	)

	BeforeEach(func() {

		azureProvider, err = provider.NewCloudProvider("azure")
		Expect(err).NotTo(HaveOccurred())
		Expect(azureProvider).ToNot(BeNil())

		test_helpers.InitializeAzureProvider(azureProvider)

		err = azureProvider.Connect()
		Expect(err).NotTo(HaveOccurred())

		azureCompute, err = azureProvider.GetCompute()
		Expect(err).NotTo(HaveOccurred())

		// ensure 2 test VMs have been created for these tests
		testInstances = test_helpers.AzureDeployTestInstances("test", 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(testInstances)).To(Equal(2))

		logger.TraceMessage("Test VMs: %# v", testInstances)
	})

	Context("Compute resources", func() {

		It("retrieves a list of compute instances", func() {

			instances, err := azureCompute.ListInstances()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(instances)).To(Equal(len(testInstances)))

			for _, instance := range instances {
				ipAddress, exists := testInstances[instance.Name()]
				Expect(exists).To(BeTrue())
				Expect(instance.PublicIP()).To(Equal(ipAddress))
			}
		})

		It("retrieves a compute instance", func() {

			instance, err := azureCompute.GetInstance("test-X")
			Expect(err).To(HaveOccurred())

			instance, err = azureCompute.GetInstance("test-0")
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
			instance0, err = azureCompute.GetInstance("test-0")
			Expect(err).NotTo(HaveOccurred())
			Expect(instance0).ToNot(BeNil())

			instance1, err = azureCompute.GetInstance("test-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(instance1).ToNot(BeNil())
		})

		It("stops and starts a compute instance", func() {

			state, err := instance0.State()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cloud.StateRunning))
			Expect(instance0.CanConnect(22)).To(BeTrue())
			Expect(instance0.CanConnect(23)).To(BeFalse())

			err = instance0.Stop()
			Expect(err).NotTo(HaveOccurred())

			state, err = instance0.State()
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cloud.StateStopped))
			Expect(instance0.CanConnect(22)).To(BeFalse())

			azureState := test_helpers.AzureInstanceState(instance0.Name())
			Expect(azureState).To(Equal("VM deallocated"))

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
