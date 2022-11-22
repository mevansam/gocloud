package cloud_test

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	oneMB  = 1024 * 1024
	fiveMB = 5 * oneMB
)

// Common Storage Tests

func testInstanceCreation(storage cloud.Storage) {

	var (
		err error
	)

	containerName1 := "test-" + uuid.New().String()
	storageInstance1, err := storage.NewInstance(containerName1)
	Expect(err).NotTo(HaveOccurred())
	Expect(storageInstance1.Name()).To(Equal(containerName1))
	defer func() {
		_ = storageInstance1.Delete()
	}()

	containerName2 := "test-" + uuid.New().String()
	storageInstance2, err := storage.NewInstance(containerName2)
	Expect(err).NotTo(HaveOccurred())
	Expect(storageInstance2.Name()).To(Equal(containerName2))
	defer func() {
		_ = storageInstance2.Delete()
	}()

	instances, err := storage.ListInstances()
	Expect(err).NotTo(HaveOccurred())
	container1Exists := false
	container2Exists := false
	for _, i := range instances {
		if i.Name() == storageInstance1.Name() {
			container1Exists = true
		}
		if i.Name() == storageInstance2.Name() {
			container2Exists = true
		}
	}
	Expect(container1Exists).To(BeTrue())
	Expect(container2Exists).To(BeTrue())

	err = storageInstance2.Delete()
	Expect(err).NotTo(HaveOccurred())

	instances, err = storage.ListInstances()
	Expect(err).NotTo(HaveOccurred())
	container1Exists = false
	container2Exists = false
	for _, i := range instances {
		if i.Name() == storageInstance1.Name() {
			container1Exists = true
		}
		if i.Name() == storageInstance2.Name() {
			container2Exists = true
		}
	}
	Expect(container1Exists).To(BeTrue())
	Expect(container2Exists).To(BeFalse())
}

func testObjectUploadAndDownload(storageInstance cloud.StorageInstance) {

	var (
		err error
		wg  sync.WaitGroup
	)

	numObjects := (rand.Intn(3) + 5) // Upload 5 to 10 objects
	objectData := make(map[string]string)

	// Upload the objects of random size
	wg.Add(numObjects)
	for i := 0; i < numObjects; i++ {
		name := fmt.Sprintf("object%d", i)
		data := utils.RandomString((rand.Intn(9) + 1) * oneMB)
		objectData[name] = data

		go func(name, data string) {
			defer wg.Done()
			defer GinkgoRecover()

			err = storageInstance.Upload(name, "text/plain", strings.NewReader(data), int64(len(data)))
			Expect(err).NotTo(HaveOccurred())
		}(name, data)
	}
	wg.Wait()

	// Validate updated object list
	objectList, err := storageInstance.ListObjects("")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(objectList)).To(Equal(numObjects))

	for i := 0; i < numObjects; i++ {
		_, exists := objectData[fmt.Sprintf("object%d", i)]
		Expect(exists).To(BeTrue())
	}

	// Download uploaded objects and verify their data
	wg.Add(numObjects)
	for i := 0; i < numObjects; i++ {

		go func(name, data string) {
			defer wg.Done()
			defer GinkgoRecover()

			var b strings.Builder
			err = storageInstance.Download(name, &b)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.String()).To(Equal(data))

		}(objectList[i], objectData[objectList[i]])
	}
	wg.Wait()

	// Delete an object
	j := rand.Intn(numObjects)
	objectToDelete := fmt.Sprintf("object%d", j)
	err = storageInstance.DeleteObject(objectToDelete)
	Expect(err).NotTo(HaveOccurred())
	objectList, err = storageInstance.ListObjects(objectToDelete)
	Expect(len(objectList)).To(BeZero())

	// Delete remaining objects
	wg.Add(numObjects - 1)
	for i := 0; i < numObjects; i++ {
		if i != j {
			go func(name string) {
				defer wg.Done()
				defer GinkgoRecover()

				err = storageInstance.DeleteObject(name)
				Expect(err).NotTo(HaveOccurred())

			}(fmt.Sprintf("object%d", i))
		}
	}
	wg.Wait()

	objectList, err = storageInstance.ListObjects("")
	Expect(len(objectList)).To(BeZero())
}

func createTestFiles(
	tmpDir string,
	tmpFiles map[string]string,
	tmpFileData map[string]string,
	blockSize int,
) {

	var (
		err error
	)

	logger.DebugMessage("Creating temp files in %s.", tmpDir)

	// create bunch of test files and map them to a hierarchical
	// path names to be used when uploading to storage
	createTestFiles := func(fromIndex, toIndex, sizeVar int, path string) {
		for i := fromIndex; i < toIndex; i++ {
			name := fmt.Sprintf("%s/file%d", path, i)
			tmpFiles[name] = fmt.Sprintf("%s/file%d", tmpDir, i)
			tmpFileData[name] = utils.RandomString((rand.Intn(sizeVar) + 1) * blockSize)
			err = os.WriteFile(tmpFiles[name], []byte(tmpFileData[name]), 0644)
			Expect(err).ToNot(HaveOccurred())
		}
	}
	createTestFiles(0, 2, 1, "aa")
	createTestFiles(2, 5, 5, "aa/bb")
	createTestFiles(5, 8, 5, "aa/bb/cc")
	createTestFiles(8, 10, 5, "aa/bb/dd")
}

func testFileUploadAndDownload(
	storageInstance cloud.StorageInstance,
	tmpFiles map[string]string,
	tmpFileData map[string]string,
) {

	var (
		err error
		wg  sync.WaitGroup
	)

	// upload all files asynchronously
	wg.Add(len(tmpFiles))
	for name, file := range tmpFiles {

		go func(name, file string) {
			defer wg.Done()
			defer GinkgoRecover()

			err = storageInstance.UploadFile(name, "text/plain", file)
			Expect(err).NotTo(HaveOccurred())
		}(name, file)
	}
	wg.Wait()

	// Validate updated object list
	objectList, err := storageInstance.ListObjects("aa/bb/")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(objectList)).To(Equal(8))

	for i := 0; i < len(objectList); i++ {
		logger.DebugMessage("Validating object '%sd' is within path request 'aa/bb/'.", objectList[i])
		Expect(objectList[i][0:6]).To(Equal("aa/bb/"))
	}

	// Download files
	wg.Add(len(tmpFiles))
	for name, file := range tmpFiles {

		go func(name, file string) {
			defer wg.Done()
			defer GinkgoRecover()

			dlFile := file + ".dl"
			err = storageInstance.DownloadFile(name, dlFile)
			Expect(err).NotTo(HaveOccurred())
		}(name, file)
	}
	wg.Wait()

	for name, file := range tmpFiles {
		dlFile := file + ".dl"
		content, err := os.ReadFile(dlFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal(tmpFileData[name]))
	}
	wg.Wait()

	// clean up
	wg.Add(len(tmpFiles))
	for name := range tmpFiles {

		go func(name string) {
			defer wg.Done()
			defer GinkgoRecover()

			err = storageInstance.DeleteObject(name)
			Expect(err).NotTo(HaveOccurred())
		}(name)
	}
	wg.Wait()
}
