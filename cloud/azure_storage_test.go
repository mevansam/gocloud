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

var _ = Describe("Azure Storage Tests", func() {

	const oneMB = 1024 * 1024

	var (
		err error

		azureProvider provider.CloudProvider
		azureStorage  cloud.Storage
	)

	BeforeEach(func() {

		azureProvider, err = provider.NewCloudProvider("azure")
		Expect(err).NotTo(HaveOccurred())
		Expect(azureProvider).ToNot(BeNil())

		test_helpers.InitializeAzureProvider(azureProvider)

		err = azureProvider.Connect()
		Expect(err).NotTo(HaveOccurred())

		azureStorage, err = azureProvider.GetStorage()
		Expect(err).NotTo(HaveOccurred())
		Expect(azureStorage).ToNot(BeNil())

		azureStorage.SetProperties(cloud.AzureStorageProperties{
			AppendBlockSize: oneMB,
			PutBlockSize:    oneMB,
		})
	})

	It("creates, lists and deletes instances", func() {
		testInstanceCreation(azureStorage)
	})

	Context("uploading and downloading data from a container", func() {

		var (
			storageInstance cloud.StorageInstance
		)

		BeforeEach(func() {
			containerName := "test-" + uuid.New().String()
			storageInstance, err = azureStorage.NewInstance(containerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(storageInstance.Name()).To(Equal(containerName))
		})

		AfterEach(func() {
			err = storageInstance.Delete()
			if err != nil {
				logger.DebugMessage(
					"Azure storage test tear down error while deleting storage instance with name '%s': %s",
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
			tmpDir, err = ioutil.TempDir("", "azurestoragetest")
			Expect(err).NotTo(HaveOccurred())

			tmpFiles = make(map[string]string)
			tmpFileData = make(map[string]string)
			createTestFiles(tmpDir, tmpFiles, tmpFileData, oneMB)

			containerName = "test-" + uuid.New().String()
			storageInstance, err = azureStorage.NewInstance(containerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(storageInstance.Name()).To(Equal(containerName))
		})

		AfterEach(func() {
			err = storageInstance.Delete()
			if err != nil {
				logger.DebugMessage(
					"Azure storage test tear down error while deleting storage instance with name '%s': %s",
					storageInstance.Name(), err.Error())
			}

			os.RemoveAll(tmpDir)
		})

		It("uploads large files with path names and validates them", func() {
			testFileUploadAndDownload(storageInstance, tmpFiles, tmpFileData)
		})
	})
})
