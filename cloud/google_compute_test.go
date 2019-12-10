package cloud_test

import (
	"github.com/mevansam/goutils/logger"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

var _ = Describe("Google Compute Tests", func() {

	var (
		err error

		googleProvider provider.CloudProvider
		googleCompute  cloud.Compute

		testInstances map[string]string
	)

	BeforeEach(func() {

		googleProvider, err = provider.NewCloudProvider("google")
		Expect(err).NotTo(HaveOccurred())
		Expect(googleProvider).ToNot(BeNil())

		test_helpers.InitializeGoogleProvider(googleProvider)

		err = googleProvider.Connect()
		Expect(err).NotTo(HaveOccurred())

		googleCompute, err = googleProvider.GetCompute()
		Expect(err).NotTo(HaveOccurred())

		// ensure 2 test VMs have been created for these tests
		testInstances = test_helpers.GoogleDeployTestInstances("test", 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(testInstances)).To(Equal(2))

		logger.TraceMessage("Test VMs: %# v", testInstances)

		googleCompute.SetProperties(cloud.GoogleComputeProperties{
			FilterLabels: map[string]string{
				"role": "cloudbuilder-test",
			},
		})
	})

	Context("Compute resources", func() {

		It("retrieves a list of compute instances", func() {

			instances, err := googleCompute.ListInstances()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(instances)).To(Equal(len(testInstances)))

			for _, instance := range instances {
				ipAddress, exists := testInstances[instance.Name()]
				Expect(exists).To(BeTrue())
				Expect(instance.PublicIP()).To(Equal(ipAddress))
			}
		})

		It("retrieves a compute instance", func() {

			instance, err := googleCompute.GetInstance("test-X")
			Expect(err).To(HaveOccurred())

			instance, err = googleCompute.GetInstance("test-0")
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
			instance0, err = googleCompute.GetInstance("test-0")
			Expect(err).NotTo(HaveOccurred())
			Expect(instance0).ToNot(BeNil())

			instance1, err = googleCompute.GetInstance("test-1")
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

			googleState := test_helpers.GoogleInstanceState(instance0.Name())
			Expect(googleState).To(Equal("TERMINATED"))

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
