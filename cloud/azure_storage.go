package cloud

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

type AzureStorageProperties struct {

	// size of block when
	// appending to a blob
	AppendBlockSize int
	// size of block when uploading
	// blocks to a blob concurrently
	PutBlockSize int64
}

type azureStorage struct {
	storageAccountName,
	resourceGroupName,
	subscriptionID string

	ctx        context.Context
	authorizer *autorest.BearerAuthorizer

	props AzureStorageProperties
}

type azureStorageInstance struct {
	name         string
	containerURL azblob.ContainerURL

	ctx   context.Context
	props *AzureStorageProperties
}

func NewAzureStorage(
	ctx context.Context,
	authorizer *autorest.BearerAuthorizer,
	storageAccountName,
	resourceGroupName,
	locationName,
	subscriptionID string,
) (Storage, error) {

	var (
		err error

		storageAccts     storage.AccountListResultIterator
		nameAvailable    storage.CheckNameAvailabilityResult
		acctCreateFuture storage.AccountsCreateFuture

		exists bool
	)

	saClient := storage.NewAccountsClient(subscriptionID)
	saClient.Authorizer = authorizer
	_ = saClient.AddToUserAgent(httpUserAgent)

	if storageAccts, err = saClient.ListByResourceGroupComplete(ctx,
		resourceGroupName,
	); err != nil {
		return nil, err
	}
	logger.TraceMessage(
		"Searching for storage account '%s' in list of accounts for resource group '%s'.",
		storageAccountName, resourceGroupName)

	// ensure default storage account exists
	// for requested storage blob container
	exists = false
	for storageAccts.NotDone() {
		if *storageAccts.Value().Name == storageAccountName {
			exists = true
			break
		}
		if err = storageAccts.NextWithContext(ctx); err != nil {
			return nil, err
		}
	}
	if !exists {
		logger.TraceMessage("Storage account '%s', was not found so creating it.", storageAccountName)

		// create storage account
		if nameAvailable, err = saClient.CheckNameAvailability(ctx,
			storage.AccountCheckNameAvailabilityParameters{
				Name: &storageAccountName,
				Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
			},
		); err != nil {
			return nil, err
		}
		if !*nameAvailable.NameAvailable {
			return nil, fmt.Errorf("storage account name '%s' not available", storageAccountName)
		}

		if acctCreateFuture, err = saClient.Create(ctx,
			resourceGroupName,
			storageAccountName,
			storage.AccountCreateParameters{
				Sku: &storage.Sku{
					Name: storage.SkuNameStandardLRS,
				},
				Kind:                              storage.KindStorageV2,
				Location:                          &locationName,
				AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
			},
		); err != nil {
			return nil, err
		}
		if err = acctCreateFuture.WaitForCompletionRef(ctx,
			saClient.BaseClient.Client,
		); err != nil {
			return nil, err
		}
	}

	return &azureStorage{
		storageAccountName: storageAccountName,
		resourceGroupName:  resourceGroupName,
		subscriptionID:     subscriptionID,

		ctx:        ctx,
		authorizer: authorizer,

		props: AzureStorageProperties{
			// https://docs.microsoft.com/en-us/rest/api/storageservices/understanding-block-blobs--append-blobs--and-page-blobs
			AppendBlockSize: 4 * 1024 * 1024,   // 4MB
			PutBlockSize:    100 * 1024 * 1024, // 100MB
		},
	}, nil
}

func (s *azureStorage) getServiceURL() (*azblob.ServiceURL, error) {

	var (
		err error

		result storage.AccountListKeysResult
	)

	saClient := storage.NewAccountsClient(s.subscriptionID)
	saClient.Authorizer = s.authorizer
	_ = saClient.AddToUserAgent(httpUserAgent)

	if result, err = saClient.ListKeys(s.ctx,
		s.resourceGroupName,
		s.storageAccountName,
		storage.ListKeyExpandKerb,
	); err != nil {
		return nil, err
	}

	c, _ := azblob.NewSharedKeyCredential(s.storageAccountName, *(((*result.Keys)[0]).Value))
	p := azblob.NewPipeline(c, azblob.PipelineOptions{
		Telemetry: azblob.TelemetryOptions{Value: httpUserAgent},
	})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", s.storageAccountName))
	serviceURL := azblob.NewServiceURL(*u, p)

	return &serviceURL, nil
}

func (s *azureStorage) newInstance(name string, containerURL azblob.ContainerURL) StorageInstance {

	return &azureStorageInstance{
		name:         name,
		containerURL: containerURL,

		ctx:   s.ctx,
		props: &s.props,
	}
}

// interface: cloud/Storage implementation

func (s *azureStorage) SetProperties(props interface{}) {

	p := props.(AzureStorageProperties)
	if p.AppendBlockSize > 0 {
		s.props.AppendBlockSize = p.AppendBlockSize
	}
	if p.PutBlockSize > 0 {
		s.props.PutBlockSize = p.PutBlockSize
	}
}

func (s *azureStorage) NewInstance(name string) (StorageInstance, error) {

	var (
		err error

		serviceURL *azblob.ServiceURL
	)

	bcClient := storage.NewBlobContainersClient(s.subscriptionID)
	bcClient.Authorizer = s.authorizer
	_ = bcClient.AddToUserAgent(httpUserAgent)

	// ensure storage blob container exists
	if _, err = bcClient.Get(s.ctx,
		s.resourceGroupName,
		s.storageAccountName,
		name,
	); err != nil {
		logger.TraceMessage(
			"Container '%s' in storage account '%s', was not found so creating it.",
			name, s.storageAccountName)

		// create blob container
		if _, err = bcClient.Create(s.ctx,
			s.resourceGroupName,
			s.storageAccountName,
			name, storage.BlobContainer{},
		); err != nil {
			return nil, err
		}
	}

	if serviceURL, err = s.getServiceURL(); err != nil {
		return nil, err
	}

	return s.newInstance(
		name,
		serviceURL.NewContainerURL(name),
	), nil
}

func (s *azureStorage) ListInstances() ([]StorageInstance, error) {

	var (
		err error

		items storage.ListContainerItemsIterator
		value storage.ListContainerItem

		serviceURL *azblob.ServiceURL
	)

	bcClient := storage.NewBlobContainersClient(s.subscriptionID)
	bcClient.Authorizer = s.authorizer
	_ = bcClient.AddToUserAgent(httpUserAgent)

	if items, err = bcClient.ListComplete(s.ctx,
		s.resourceGroupName,
		s.storageAccountName,
		"", "", "",
	); err != nil {
		return nil, err
	}
	if serviceURL, err = s.getServiceURL(); err != nil {
		return nil, err
	}

	instances := []StorageInstance{}
	for items.NotDone() {
		value = items.Value()
		instances = append(instances,
			s.newInstance(
				*value.Name,
				serviceURL.NewContainerURL(*value.Name),
			),
		)
		if err = items.NextWithContext(s.ctx); err != nil {
			return nil, err
		}
	}

	return instances, nil
}

// interface: cloud/StorageInstance implementation

func (s *azureStorageInstance) Name() string {
	return s.name
}

func (s *azureStorageInstance) Delete() error {
	var (
		err error
	)
	logger.TraceMessage("Deleting container '%s' with URL %s.", s.name, s.containerURL.String())

	if _, err = s.containerURL.Delete(s.ctx, azblob.ContainerAccessConditions{}); err != nil {
		return err
	}
	for {
		_, err = s.containerURL.GetProperties(s.ctx, azblob.LeaseAccessConditions{})
		if err != nil {
			if stErr, ok := err.(azblob.StorageError); ok {
				if stErr.ServiceCode() == azblob.ServiceCodeType("ContainerNotFound") {
					break
				}
			}
			return err
		}
		logger.TraceMessage("Waiting for container '%s' to be deleted.", s.name)
	}
	return nil
}

func (s *azureStorageInstance) ListObjects(path string) ([]string, error) {

	var (
		err error

		blobListResponse *azblob.ListBlobsFlatSegmentResponse
	)

	blobList := []string{}
	marker := azblob.Marker{}

	for {
		if blobListResponse, err = s.containerURL.ListBlobsFlatSegment(
			s.ctx,
			marker,
			azblob.ListBlobsSegmentOptions{
				Prefix: path,
				Details: azblob.BlobListingDetails{
					Snapshots: true,
				},
			}); err != nil {
			return []string{}, err
		}
		logger.TraceMessage(
			"Retrieved list of objects in container '%s' filtered by path '%s': %# v",
			s.name, path, blobListResponse.Segment.BlobItems)

		for _, item := range blobListResponse.Segment.BlobItems {
			blobList = append(blobList, item.Name)
		}
		if blobListResponse.NextMarker.Val == nil ||
			*blobListResponse.NextMarker.Val == "" {
			break
		}
		marker.Val = blobListResponse.NextMarker.Val
	}

	return blobList, nil
}

func (s *azureStorageInstance) DeleteObject(name string) error {

	var (
		err error
	)

	logger.TraceMessage(
		"Deleting blob with name '%s' in container '%s'.",
		name, s.name)

	blobURL := s.containerURL.NewBlobURL(name)
	_, err = blobURL.Delete(s.ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	return err
}

func (s *azureStorageInstance) Upload(name, contentType string, data io.Reader, size int64) error {

	var (
		err error

		n   int
		b   []byte
		eof bool
	)

	blobURL := s.containerURL.NewAppendBlobURL(name)
	logger.TraceMessage(
		"Uploading blob with name '%s' of size %d to container '%s'.",
		name, size, s.name)

	if _, err = blobURL.Create(s.ctx,
		azblob.BlobHTTPHeaders{
			ContentType: contentType,
		},
		azblob.Metadata{},
		azblob.BlobAccessConditions{},
		azblob.BlobTagsMap{},
		azblob.ClientProvidedKeyOptions{},
	); err != nil {
		return err
	}

	b = make([]byte, s.props.AppendBlockSize)
	eof = false
	for !eof {
		if n, err = data.Read(b); err != nil {
			if err != io.EOF {
				return err
			} else {
				eof = true
			}
		}
		if n > 0 {
			logger.TraceMessage(
				"Appending block of size %d to blob with name '%s' in container '%s'.",
				n, name, s.name)

			if _, err = blobURL.AppendBlock(s.ctx,
				bytes.NewReader(b[0:n]),
				azblob.AppendBlobAccessConditions{},
				nil,
				azblob.ClientProvidedKeyOptions{},
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *azureStorageInstance) UploadFile(name, contentType, path string) error {

	var (
		err error

		file     *os.File
		fileInfo os.FileInfo

		wg sync.WaitGroup

		blockList *azblob.BlockList

		i int64
	)

	if file, err = os.Open(path); err != nil {
		return err
	}
	defer file.Close()

	if fileInfo, err = file.Stat(); err != nil {
		return err
	}
	size := fileInfo.Size()

	blobURL := s.containerURL.NewBlockBlobURL(name)
	logger.TraceMessage(
		"Uploading file blob with name '%s' of size %d to container '%s'.",
		name, size, s.name)

	if size > s.props.PutBlockSize {

		numBlocks := size / s.props.PutBlockSize
		partialBlockSize := size % s.props.PutBlockSize
		if partialBlockSize > 0 {
			numBlocks++
		}

		hasErrors := false
		errors := make([]error, numBlocks)

		wg.Add(int(numBlocks))
		for i = 0; i < numBlocks; i++ {

			go func(blockNum int64) {
				defer wg.Done()

				logger.TraceMessage(
					"Putting block %d of blob to blob with name '%s' in container '%s'.",
					blockNum, name, s.name)

				blockID := make([]byte, 8)
				binary.LittleEndian.PutUint64(blockID, uint64(blockNum))

				if _, err = blobURL.StageBlock(s.ctx,
					base64.StdEncoding.EncodeToString(blockID),
					utils.NewChunkReadSeeker(
						file,
						blockNum * s.props.PutBlockSize,
						s.props.PutBlockSize,
					),
					azblob.LeaseAccessConditions{},
					nil,
					azblob.ClientProvidedKeyOptions{},
				); err != nil {
					hasErrors = true
					errors[blockNum] = err
				}
			}(i)
		}
		wg.Wait()

		if hasErrors {
			var errMsg strings.Builder
			for i = 0; i < numBlocks; i++ {
				if errors[i] != nil {
					errMsg.WriteString(
						fmt.Sprintf("Uploading block %d failed: %s; ", i, errors[i].Error()),
					)
				}
			}
			return fmt.Errorf(errMsg.String())
		}

		if blockList, err = blobURL.GetBlockList(s.ctx,
			azblob.BlockListUncommitted,
			azblob.LeaseAccessConditions{},
		); err != nil {
			return err
		}
		logger.TraceMessage(
			"Commiting %d uncommited blocks of blob '%s' in container '%s'.",
			len(blockList.UncommittedBlocks), name, s.name)

		ids := []string{}
		for _, u := range blockList.UncommittedBlocks {
			ids = append(ids, u.Name)
		}
		_, err = blobURL.CommitBlockList(s.ctx,
			ids,
			azblob.BlobHTTPHeaders{
				ContentType: contentType,
			},
			azblob.Metadata{},
			azblob.BlobAccessConditions{},
			azblob.AccessTierHot,
			azblob.BlobTagsMap{},
			azblob.ClientProvidedKeyOptions{},
		)

	} else {
		logger.TraceMessage(
			"Creating block blob of size %d to blob with name '%s' in container '%s'.",
			size, name, s.name)

		_, err = blobURL.Upload(s.ctx,
			file,
			azblob.BlobHTTPHeaders{
				ContentType: contentType,
			},
			azblob.Metadata{},
			azblob.BlobAccessConditions{},
			azblob.AccessTierHot,
			azblob.BlobTagsMap{},
			azblob.ClientProvidedKeyOptions{},
		)
	}

	return err
}

func (s *azureStorageInstance) Download(name string, data io.Writer) error {

	var (
		err error

		resp *azblob.DownloadResponse
	)

	blobURL := s.containerURL.NewBlobURL(name)
	logger.TraceMessage(
		"Downloading blob with name '%s' from container '%s' having URL '%s'.",
		name, s.name, blobURL.String())

	if resp, err = blobURL.Download(s.ctx,
		0, azblob.CountToEnd,
		azblob.BlobAccessConditions{},
		false,
		azblob.ClientProvidedKeyOptions{},
	); err != nil {
		return err
	}
	defer resp.Response().Body.Close()
	body := resp.Body(azblob.RetryReaderOptions{})

	_, err = io.CopyBuffer(
		data,
		body,
		make([]byte, s.props.AppendBlockSize),
	)
	return err
}

func (s *azureStorageInstance) DownloadFile(name, path string) error {

	var (
		err error

		file *os.File

		wg     *sync.WaitGroup
		size   int64
		errors []error
	)

	if file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644); err != nil {
		return err
	}
	defer file.Close()

	wg, size, errors, err = s.DownloadAsync(name, file)
	wg.Wait()

	if err != nil {
		var errMsg strings.Builder
		errMsg.WriteString(err.Error())

		for i := 0; i < len(errors); i++ {
			if errors[i] != nil {
				errMsg.WriteString(
					fmt.Sprintf("; Uploading block %d failed: %s", i, errors[i].Error()),
				)
			}
		}
		return fmt.Errorf(errMsg.String())
	}

	err = file.Truncate(size)
	return err
}

func (s *azureStorageInstance) DownloadAsync(name string, data io.WriterAt) (*sync.WaitGroup, int64, []error, error) {

	var (
		err error
		wg  sync.WaitGroup

		blobListResponse *azblob.ListBlobsFlatSegmentResponse

		i int64
	)

	// get size of blob to download
	if blobListResponse, err = s.containerURL.ListBlobsFlatSegment(
		s.ctx,
		azblob.Marker{},
		azblob.ListBlobsSegmentOptions{
			Prefix: name,
			Details: azblob.BlobListingDetails{
				Snapshots: true,
			},
		}); err != nil {
		return nil, 0, nil, err
	}
	numBlobs := len(blobListResponse.Segment.BlobItems)
	if numBlobs == 0 {
		return nil, 0, nil, fmt.Errorf("blob named '%s' not found in container '%s'", name, s.name)
	}
	if numBlobs > 1 {
		return nil, 0, nil, fmt.Errorf("found more than one blob named '%s' in container '%s'", name, s.name)
	}
	size := *blobListResponse.Segment.BlobItems[0].Properties.ContentLength

	blobURL := s.containerURL.NewBlobURL(name)
	logger.TraceMessage(
		"Downloading blob with name '%s' of size %d from container '%s'.",
		name, size, s.name)

	numBlocks := size / s.props.PutBlockSize
	partialBlockSize := size % s.props.PutBlockSize
	if partialBlockSize > 0 {
		numBlocks++
	}

	hasErrors := false
	errors := make([]error, numBlocks)

	wg.Add(int(numBlocks))
	for i = 0; i < numBlocks; i++ {

		go func(blockNum int64) {
			defer wg.Done()

			var (
				resp *azblob.DownloadResponse

				n   int
				b   []byte
				eof bool
			)

			logger.TraceMessage(
				"Downloading block %d of blob with name '%s' in container '%s'.",
				blockNum, name, s.name)

			offset := blockNum * s.props.PutBlockSize
			end := (blockNum+1) * s.props.PutBlockSize
			if end > size {
				end = size
			}

			if resp, err = blobURL.Download(s.ctx,
				offset, end,
				azblob.BlobAccessConditions{},
				false,
				azblob.ClientProvidedKeyOptions{},
			); err != nil {
				errors[blockNum] = err
				hasErrors = true
				return
			}
			defer resp.Response().Body.Close()
			body := resp.Body(azblob.RetryReaderOptions{})

			b = make([]byte, s.props.PutBlockSize)
			eof = false
			for !eof {
				if n, err = body.Read(b); err != nil {
					if err != io.EOF {
						errors[blockNum] = err
						hasErrors = true
						return
					} else {
						eof = true
					}
				}
				if n > 0 {
					if _, err = data.WriteAt(b[0:n], offset); err != nil {
						errors[blockNum] = err
						hasErrors = true
						return
					}
					offset = offset + int64(n)
				}
			}
		}(i)
	}

	if hasErrors {
		return &wg, size, errors,
			fmt.Errorf("failed to download blob '%s' from container '%s'", name, s.name)
	} else {
		return &wg, size, nil, nil
	}
}
