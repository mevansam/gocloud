package cloud

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/google/uuid"

	"github.com/mevansam/goutils/logger"
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

	ctx         context.Context	
	clientCreds *azidentity.ClientSecretCredential
	clientOpts  *arm.ClientOptions

	props AzureStorageProperties
}

type azureStorageInstance struct {
	name,
	storageURL,
	containerURL string

	props *AzureStorageProperties

	storage *azureStorage
}

const storageURLF = `https://%s.blob.core.windows.net`

func NewAzureStorage(
	ctx context.Context,
	clientCreds *azidentity.ClientSecretCredential,
	clientOpts *arm.ClientOptions,
	storageAccountName,
	resourceGroupName,
	locationName,
	subscriptionID,
	servicePrincipalID string,
) (Storage, error) {

	var (
		err error

		acctsClient   *armstorage.AccountsClient
		roleDefClient *armauthorization.RoleDefinitionsClient
		roleAssClient *armauthorization.RoleAssignmentsClient

		salResp armstorage.AccountsClientListByResourceGroupResponse
		naResp  armstorage.AccountsClientCheckNameAvailabilityResponse
		rdResp  armauthorization.RoleDefinitionsClientListResponse

		pCreateResp *azruntime.Poller[armstorage.AccountsClientCreateResponse]
		createResp  armstorage.AccountsClientCreateResponse

		exists bool
	)

	logger.TraceMessage(
		"Searching for storage account '%s' in list of accounts for resource group '%s'.",
		storageAccountName, resourceGroupName)

	if acctsClient, err = armstorage.NewAccountsClient(subscriptionID, clientCreds, clientOpts); err != nil {
		return nil, err
	}
	listSAs := acctsClient.NewListByResourceGroupPager(resourceGroupName, nil)

	// ensure default storage account exists
	// for requested storage blob container
	exists = false
	for listSAs.More() {
		if salResp, err = listSAs.NextPage(ctx); err != nil {
			return nil, err
		}

		for _, sa := range salResp.AccountListResult.Value {
			if *sa.Name == storageAccountName {
				exists = true
				break
			}
		}
	}

	if !exists {
		logger.TraceMessage("Storage account '%s', was not found so creating it.", storageAccountName)

		// create storage account
		if naResp, err = acctsClient.CheckNameAvailability(ctx,
			armstorage.AccountCheckNameAvailabilityParameters{
				Name: &storageAccountName,
				Type: to.Ptr("Microsoft.Storage/storageAccounts"),
			},
			nil,
		); err != nil {
			return nil, err
		}
		if !*naResp.NameAvailable {
			return nil, fmt.Errorf("storage account name '%s' not available", storageAccountName)
		}

		if pCreateResp, err = acctsClient.BeginCreate(ctx,
			resourceGroupName,
			storageAccountName,
			armstorage.AccountCreateParameters{
				SKU: &armstorage.SKU{
					Name: to.Ptr(armstorage.SKUNameStandardLRS),
				},
				Kind:       to.Ptr(armstorage.KindStorageV2),
				Location:   &locationName,
				Properties: &armstorage.AccountPropertiesCreateParameters{
					AllowBlobPublicAccess: to.Ptr(false),
					AccessTier: to.Ptr(armstorage.AccessTierHot),
				},
			},
			nil,
		); err != nil {
			return nil, err
		}
		if createResp, err = pCreateResp.PollUntilDone(ctx, nil); err != nil {
			return nil, err
		}

		// NB: to see role assignment names run
		// 'az role definition list --query "sort_by([?contains(roleName, 'Storage')].{Name:roleName,Id:name}, &Name)" --output table'
		// 
		// assign role to permit blob data upload/download
		if roleDefClient, err = armauthorization.NewRoleDefinitionsClient(
			clientCreds, 
			clientOpts,
		); err != nil {
			return nil, err
		}
		pRoleDefList := roleDefClient.NewListPager(
			*createResp.ID, 
			&armauthorization.RoleDefinitionsClientListOptions{
				Filter: to.Ptr("roleName eq 'Storage Blob Data Contributor'"),
			},
		)
		if rdResp, err = pRoleDefList.NextPage(ctx); err != nil {
			return nil, err
		}
		if len(rdResp.Value) == 0 {
			return nil, fmt.Errorf("Unable to determine role definition ID for 'Storage Blob Data Contributor' needed for blob upload/download")
		}		

		if roleAssClient, err = armauthorization.NewRoleAssignmentsClient(
			subscriptionID, 
			clientCreds, 
			clientOpts,
		); err != nil {
			return nil, err
		}
		if _, err = roleAssClient.Create(
			ctx, 
			*createResp.ID, 
			uuid.New().String(),
			armauthorization.RoleAssignmentCreateParameters{
				Properties: &armauthorization.RoleAssignmentProperties{
					PrincipalID: to.Ptr(servicePrincipalID),
					RoleDefinitionID: rdResp.Value[0].ID,
				},
			},
			nil,
		); err != nil {
			return nil, err
		}
	}

	return &azureStorage{
		storageAccountName: storageAccountName,
		resourceGroupName:  resourceGroupName,
		subscriptionID:     subscriptionID,

		ctx:         ctx,
		clientCreds: clientCreds,
		clientOpts:  clientOpts,

		props: AzureStorageProperties{
			// https://docs.microsoft.com/en-us/rest/api/storageservices/understanding-block-blobs--append-blobs--and-page-blobs
			AppendBlockSize: 4 * 1024 * 1024,   // 4MB
			PutBlockSize:    100 * 1024 * 1024, // 100MB
		},
	}, nil
}

func (s *azureStorage) newInstance(name string) (StorageInstance, error) {

	storageURL := fmt.Sprintf(storageURLF, s.storageAccountName)

	return &azureStorageInstance{
		name: name,		
		storageURL: storageURL,
		containerURL: strings.Join([]string{storageURL, name}, "/"),

		props: &s.props,

		storage: s,
	}, nil
}

func (s *azureStorage) deleteInstance(name string) error {

	var (
		err error

		client *armstorage.BlobContainersClient
	)

	if client, err = armstorage.NewBlobContainersClient(s.subscriptionID, s.clientCreds, s.clientOpts); err != nil {
		return err
	}

	if _, err = client.Delete(s.ctx,
		s.resourceGroupName,
		s.storageAccountName,
		name, 
		nil,
	); err != nil {
		return err
	}
	for {
		if _, err = client.Get(s.ctx,
			s.resourceGroupName,
			s.storageAccountName,
			name,
			nil,
		); err != nil {
			if !strings.Contains(err.Error(), "ERROR CODE: ContainerNotFound") {
				logger.ErrorMessage("Container delete request returned an err: %s", err.Error())
			}
			break
		}
		logger.TraceMessage("Waiting for container '%s' to be deleted.", name)
	}
	return nil
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

		client *armstorage.BlobContainersClient
	)

	if client, err = armstorage.NewBlobContainersClient(s.subscriptionID, s.clientCreds, s.clientOpts); err != nil {
		return nil, err
	}

	// ensure storage blob container exists
	if _, err = client.Get(s.ctx,
		s.resourceGroupName,
		s.storageAccountName,
		name,
		nil,
	); err != nil {
		logger.TraceMessage(
			"Container '%s' in storage account '%s', was not found so creating it.",
			name, s.storageAccountName)

		// create blob container
		if _, err = client.Create(s.ctx,
			s.resourceGroupName,
			s.storageAccountName,
			name, 
			armstorage.BlobContainer{},
			nil,
		); err != nil {
			return nil, err
		}
	}

	return s.newInstance(name)
}

func (s *azureStorage) ListInstances() ([]StorageInstance, error) {

	var (
		err error

		client *armstorage.BlobContainersClient
		resp   armstorage.BlobContainersClientListResponse

		instance StorageInstance
	)

	if client, err = armstorage.NewBlobContainersClient(s.subscriptionID, s.clientCreds, s.clientOpts); err != nil {
		return nil, err
	}
	listContainers := client.NewListPager(s.resourceGroupName, s.storageAccountName, nil)

	instances := []StorageInstance{}
	for listContainers.More() {
		if resp, err = listContainers.NextPage(s.ctx); err != nil {
			return nil, err
		}
		for _, container := range resp.Value {
			if instance, err = s.newInstance(*container.Name); err != nil {
				return nil, err
			}
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// interface: cloud/StorageInstance implementation

func (s *azureStorageInstance) Name() string {
	return s.name
}

func (s *azureStorageInstance) Delete() error {
	return s.storage.deleteInstance(s.name)
}

func (s *azureStorageInstance) ListObjects(path string) ([]string, error) {

	var (
		err error

		client *azblob.Client
		resp   azblob.ListBlobsFlatResponse
	)

	if client, err = azblob.NewClient(
		s.storageURL, 
		s.storage.clientCreds, 
		&azblob.ClientOptions{
			ClientOptions: s.storage.clientOpts.ClientOptions,
		},
	); err != nil {
		return []string{}, err
	}

	blobList := []string{}
	list := client.NewListBlobsFlatPager(s.name, &azblob.ListBlobsFlatOptions{
		Prefix: &path,
		Include: azblob.ListBlobsInclude{
			Snapshots: true,
		},
	})
	for list.More() {
		if resp, err = list.NextPage(s.storage.ctx); err != nil {
			return []string{}, err
		}
		logger.TraceMessage(
			"Retrieved list of objects in container '%s' filtered by path '%s': %# v",
			s.name, path, resp.Segment.BlobItems)

		for _, item := range resp.Segment.BlobItems {
			blobList = append(blobList, *item.Name)
		}
	}

	return blobList, nil
}

func (s *azureStorageInstance) DeleteObject(name string) error {

	var (
		err error

		client *azblob.Client
	)

	if client, err = azblob.NewClient(
		s.storageURL, 
		s.storage.clientCreds, 
		&azblob.ClientOptions{
			ClientOptions: s.storage.clientOpts.ClientOptions,
		},
	); err != nil {
		return err
	}
	
	logger.TraceMessage(
		"Deleting blob with name '%s' in container '%s'.",
		name, s.name)

	if _, err = client.DeleteBlob(
		s.storage.ctx, 
		s.name, 
		name, 
		&azblob.DeleteBlobOptions{
			DeleteSnapshots: to.Ptr(azblob.DeleteSnapshotsOptionTypeInclude),
		},
	); err != nil {
		return err
	}

	return err
}

func (s *azureStorageInstance) Upload(name, contentType string, data io.Reader, size int64) error {

	var (
		err error

		client *azblob.Client
	)

	if client, err = azblob.NewClient(
		s.storageURL, 
		s.storage.clientCreds, 
		&azblob.ClientOptions{
			ClientOptions: s.storage.clientOpts.ClientOptions,
		},
	); err != nil {
		return err
	}

	_, err = client.UploadStream(
		s.storage.ctx, 
		s.name, 
		name, 
		data, 
		&blockblob.UploadStreamOptions{
			BlockSize: int64(s.props.AppendBlockSize),
			Concurrency: runtime.NumCPU(),
			HTTPHeaders: &blob.HTTPHeaders{
				BlobContentType: &contentType,
			},
		},
	)
	return err
}

func (s *azureStorageInstance) UploadFile(name, contentType, path string) error {

	var (
		err error

		file   *os.File
		client *azblob.Client
	)

	if file, err = os.Open(path); err != nil {
		return err
	}
	defer file.Close()

	if client, err = azblob.NewClient(
		s.storageURL, 
		s.storage.clientCreds, 
		&azblob.ClientOptions{
			ClientOptions: s.storage.clientOpts.ClientOptions,
		},
	); err != nil {
		return err
	}

	_, err = client.UploadFile(
		s.storage.ctx, 
		s.name, 
		name, 
		file,
		&azblob.UploadFileOptions{
			BlockSize: s.props.PutBlockSize,
			Concurrency: uint16(runtime.NumCPU()),
			HTTPHeaders: &blob.HTTPHeaders{
				BlobContentType: &contentType,
			},
		},
	)
	return err
}

func (s *azureStorageInstance) Download(name string, data io.Writer) error {

	var (
		err error

		client *azblob.Client
		resp   azblob.DownloadStreamResponse
	)

	if client, err = azblob.NewClient(
		s.storageURL, 
		s.storage.clientCreds, 
		&azblob.ClientOptions{
			ClientOptions: s.storage.clientOpts.ClientOptions,
		},
	); err != nil {
		return err
	}

	if resp, err = client.DownloadStream(
		s.storage.ctx, 
		s.name, 
		name, 
		&azblob.DownloadStreamOptions{},
	); err != nil {
		return err
	}

	len, err := io.CopyBuffer(
		data,
		resp.Body,
		make([]byte, s.props.AppendBlockSize),
	)
	logger.TraceMessage(
		"Downloaded %d bytes for blob %s/%s/%s.",
		len, s.storageURL, s.name, name, 
	)
	return err
}

func (s *azureStorageInstance) DownloadFile(name, path string) error {

	var (
		err error

		file   *os.File
		client *azblob.Client
	)

	if file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644); err != nil {
		return err
	}
	defer file.Close()

	if client, err = azblob.NewClient(
		s.storageURL, 
		s.storage.clientCreds, 
		&azblob.ClientOptions{
			ClientOptions: s.storage.clientOpts.ClientOptions,
		},
	); err != nil {
		return err
	}

	_, err = client.DownloadFile(
		s.storage.ctx,
		s.name,
		name,
		file,
		&azblob.DownloadFileOptions{
			BlockSize: int64(s.props.AppendBlockSize),
			Concurrency: uint16(runtime.NumCPU()),
		},
	)
	return err
}
