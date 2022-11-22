package cloud_test

import (
	"os"

	"github.com/google/uuid"

	"github.com/mevansam/goutils/logger"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

var _ = Describe("Google Storage Tests", func() {

	var (
		err error

		googleProvider provider.CloudProvider
		googleStorage  cloud.Storage
	)

	BeforeEach(func() {

		googleProvider, err = provider.NewCloudProvider("google")
		Expect(err).NotTo(HaveOccurred())
		Expect(googleProvider).ToNot(BeNil())

		test_helpers.InitializeGoogleProvider(googleProvider)

		err = googleProvider.Connect()
		Expect(err).NotTo(HaveOccurred())

		googleStorage, err = googleProvider.GetStorage()
		Expect(err).NotTo(HaveOccurred())
		Expect(googleStorage).ToNot(BeNil())

		googleStorage.SetProperties(cloud.GoogleStorageProperties{
			// BlockSize: fiveMB,
		})
	})

	It("creates, lists and deletes instances", func() {
		testInstanceCreation(googleStorage)
	})

	Context("uploading and downloading data from a container", func() {

		var (
			storageInstance cloud.StorageInstance
		)

		BeforeEach(func() {
			containerName := "test-" + uuid.New().String()
			storageInstance, err = googleStorage.NewInstance(containerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(storageInstance.Name()).To(Equal(containerName))
		})

		AfterEach(func() {
			err = storageInstance.Delete()
			if err != nil {
				logger.DebugMessage(
					"Google storage test tear down error while deleting storage instance with name '%s': %s",
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
			tmpDir, err = os.MkdirTemp("", "googlestoragetest")
			Expect(err).NotTo(HaveOccurred())

			tmpFiles = make(map[string]string)
			tmpFileData = make(map[string]string)
			createTestFiles(tmpDir, tmpFiles, tmpFileData, fiveMB)

			containerName = "test-" + uuid.New().String()
			storageInstance, err = googleStorage.NewInstance(containerName)
			Expect(err).NotTo(HaveOccurred())
			Expect(storageInstance.Name()).To(Equal(containerName))
		})

		AfterEach(func() {
			err = storageInstance.Delete()
			if err != nil {
				logger.DebugMessage(
					"Google storage test tear down error while deleting storage instance with name '%s': %s",
					storageInstance.Name(), err.Error())
			}

			os.RemoveAll(tmpDir)
		})

		It("uploads large files with path names and validates them", func() {
			testFileUploadAndDownload(storageInstance, tmpFiles, tmpFileData)
		})
	})
})
