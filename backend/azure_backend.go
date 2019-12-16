package backend

import (
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"

	forms_config "github.com/mevansam/gocloud/forms"
	"github.com/mevansam/gocloud/provider"
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

	if _, err = form.NewInputField(
		/* name */ "resource_group_name",
		/* displayName */ "Resource Group Name",
		/* description */ "The Azure resource group name where storage resources will be created.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{},
	); err != nil {
		return err
	}
	if _, err = form.NewInputField(
		/* name */ "storage_account_name",
		/* displayName */ "Storage Account Name",
		/* description */ "The name of the storage account to use for the state container.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{},
	); err != nil {
		return err
	}
	if _, err = form.NewInputField(
		/* name */ "container_name",
		/* displayName */ "Container Name",
		/* description */ "The name of the storage container where state will be saved.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{},
	); err != nil {
		return err
	}
	if _, err = form.NewInputField(
		/* name */ "key",
		/* displayName */ "Key",
		/* description */ "The key with which to identify the state blob in the container.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{},
	); err != nil {
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

	*configCopy = *config

	return copy, nil
}

func (b *azurermBackend) IsValid() bool {

	config := b.cloudBackend.
		config.(*azurermBackendConfig)

	return config.ResourceGroupName != nil && len(*config.ResourceGroupName) > 0 &&
		config.StorageAccountName != nil && len(*config.StorageAccountName) > 0 &&
		config.ContainerName != nil && len(*config.ContainerName) > 0 &&
		config.Key != nil && len(*config.Key) > 0
	return false
}

// interface: backend/CloudBackend functions

func (b *azurermBackend) Initialize(provider provider.CloudProvider) error {
	return nil
}
