package cloud_test

import (
	"io/ioutil"
	"os"

	"github.com/google/uuid"

	"github.com/mevansam/goutils/logger"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

var _ = Describe("AWS Storage Tests", func() {

	var (
		err error

		awsProvider provider.CloudProvider
		awsStorage  cloud.Storage
	)

	BeforeEach(func() {

		awsProvider, err = provider.NewCloudProvider("aws")
		Expect(err).NotTo(HaveOccurred())
		Expect(awsProvider).ToNot(BeNil())

		test_helpers.InitializeAWSProvider(awsProvider)

		err = awsProvider.Connect()
		Expect(err).NotTo(HaveOccurred())

		awsStorage, err = awsProvider.GetStorage()
		Expect(err).NotTo(HaveOccurred())
		Expect(awsStorage).ToNot(BeNil())

		awsStorage.SetProperties(cloud.AWSStorageProperties{
			BlockSize: fiveMB,
		})
	})

	It("creates, lists and deletes instances", func() {
		testInstanceCreation(awsStorage)
	})

	Context("uploading and downloading data from a container", func() {

		var (
			storageInstance cloud.StorageInstance
		)

		BeforeEach(func() {
			containerName := "test-" + uuid.New().String()
			storageInstance, err = awsStorage.NewInstance(containerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(storageInstance.Name()).To(Equal(containerName))
		})

		AfterEach(func() {
			err = storageInstance.Delete()
			if err != nil {
				logger.DebugMessage(
					"AWS storage test tear down error while deleting storage instance with name '%s': %s",
					storageInstance.Name(), err.Error())
			}
		})

		It("uploads a few blobs and validates them", func() {
			testObjectUploadAndDownload(storageInstance)
		})
	})

	Context("uploading and downloading files from a container", func() {

		var (
			tmpDir      string
			tmpFiles    map[string]string
			tmpFileData map[string]string

			containerName   string
			storageInstance cloud.StorageInstance
		)

		BeforeEach(func() {
			tmpDir, err = ioutil.TempDir("", "awsstoragetest")
			Expect(err).NotTo(HaveOccurred())

			tmpFiles = make(map[string]string)
			tmpFileData = make(map[string]string)
			createTestFiles(tmpDir, tmpFiles, tmpFileData, fiveMB)

			containerName = "test-" + uuid.New().String()
			storageInstance, err = awsStorage.NewInstance(containerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(storageInstance.Name()).To(Equal(containerName))
		})

		AfterEach(func() {
			err = storageInstance.Delete()
			if err != nil {
				logger.DebugMessage(
					"AWS storage test tear down error while deleting storage instance with name '%s': %s",
					storageInstance.Name(), err.Error())
			}

			os.RemoveAll(tmpDir)
		})

		It("uploads large files with path names and validates them", func() {
			testFileUploadAndDownload(storageInstance, tmpFiles, tmpFileData)
		})
	})
})
