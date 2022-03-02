package backend

import (
	"fmt"
	"strings"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"

	forms_config "github.com/mevansam/gocloud/forms"
)

type azurermBackend struct {
	cloudBackend

	isInitialized bool
}

type azurermBackendConfig struct {
	ResourceGroupName  *string `json:"resource_group_name,omitempty" form_field:"resource_group_name"`
	StorageAccountName *string `json:"storage_account_name,omitempty" form_field:"storage_account_name"`
	ContainerName      *string `json:"container_name,omitempty" form_field:"container_name"`
	Key                *string `json:"key,omitempty" form_field:"key"`
}

func newAzureRMBackend() (CloudBackend, error) {

	var (
		err           error
		beckendConfig azurermBackendConfig
	)

	backend := &azurermBackend{
		cloudBackend: cloudBackend{
			name:         "azurerm",
			providerType: "azure",

			config: &beckendConfig,
		},
		isInitialized: false,
	}
	err = backend.createAzureRMInputForm()
	return backend, err
}

func (b *azurermBackend) createAzureRMInputForm() error {

	// Do not recreate form template if it exists
	clougConfig := forms_config.CloudConfigForms
	if clougConfig.HasGroup(b.name) {
		return nil
	}

	var (
		err  error
		form *forms.InputGroup
	)

	form = forms_config.CloudConfigForms.NewGroup(b.name, "Azure Resource Manager Storage Backend")

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "resource_group_name",
		DisplayName: "Resource Group Name",
		Description: "The Azure resource group name where storage resources will be created.",
		InputType:   forms.String,
		Tags:        []string{"backend"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "storage_account_name",
		DisplayName: "Storage Account Name",
		Description: "The name of the storage account to use for the state container.",
		InputType:   forms.String,
		Tags:        []string{"backend"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "container_name",
		DisplayName: "Container Name",
		Description: "The name of the storage container where state will be saved.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target-undeployed"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "key",
		DisplayName: "Key",
		Description: "The key with which to identify the state blob in the container.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target-undeployed"},
	}); err != nil {
		return err
	}

	return nil
}

// interface: config/Configurable functions of base cloud backend

func (b *azurermBackend) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudBackend
	)

	if copy, err = newAzureRMBackend(); err != nil {
		return nil, err
	}

	config := b.cloudBackend.
		config.(*azurermBackendConfig)
	configCopy := copy.(*azurermBackend).cloudBackend.
		config.(*azurermBackendConfig)

	configCopy.ResourceGroupName = utils.CopyStrPtr(config.ResourceGroupName)
	configCopy.StorageAccountName = utils.CopyStrPtr(config.StorageAccountName)
	configCopy.ContainerName = utils.CopyStrPtr(config.ContainerName)
	configCopy.Key = utils.CopyStrPtr(config.Key)

	return copy, nil
}

func (b *azurermBackend) IsValid() bool {

	config := b.cloudBackend.
		config.(*azurermBackendConfig)

	return config.ResourceGroupName != nil && len(*config.ResourceGroupName) > 0 &&
		config.StorageAccountName != nil && len(*config.StorageAccountName) > 0 &&
		config.ContainerName != nil && len(*config.ContainerName) > 0 &&
		config.Key != nil && len(*config.Key) > 0
}

// interface: backend/CloudBackend functions

func (b *azurermBackend) Configure(
	cloudProvider provider.CloudProvider,
	storagePrefix, stateKey string,
) error {

	var (
		err error

		inputForm forms.InputForm

		defaultResourceGroup *string
		defaultLocation      *string
	)

	if cloudProvider.Name() != b.providerType {
		return fmt.Errorf("the azurerm backend can only be used with an azure cloud provider")
	}
	if inputForm, err = cloudProvider.InputForm(); err != nil {
		return err
	}
	if defaultResourceGroup, err = inputForm.GetFieldValue("default_resource_group"); err != nil {
		return err
	}
	if defaultResourceGroup == nil {
		return fmt.Errorf("azure provider's resource group cannot be empty")
	}
	if defaultLocation, err = inputForm.GetFieldValue("default_location"); err != nil {
		return err
	}
	if defaultLocation == nil {
		return fmt.Errorf("azure provider's location cannot be empty")
	}

	storageAccountName := provider.GetAzureStorageAccountName(cloudProvider)
	containerName := strings.ToLower(
		fmt.Sprintf("%s-%s", storagePrefix, *defaultLocation),
	)

	config := b.cloudBackend.
		config.(*azurermBackendConfig)
	config.ResourceGroupName = defaultResourceGroup
	config.StorageAccountName = &storageAccountName
	config.ContainerName = &containerName
	config.Key = &stateKey

	// rebind fields
	if _, err = b.InputForm(); err != nil {
		return err
	}
	logger.TraceMessage(
		"AzureRM backend configured using provider attributes: %# v",
		config)

	return nil
}

func (b *azurermBackend) GetStorageInstanceName() string {
	return *b.cloudBackend.config.(*azurermBackendConfig).ContainerName
}
