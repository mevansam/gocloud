package backend

import (
	"encoding/json"
	"fmt"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"

	forms_config "github.com/mevansam/gocloud/forms"
)

type CloudBackend interface {
	config.Configurable

	// configures the storage for this backend with common
	// attributes fetched from a compatible provider
	Configure(
		cloudProvider provider.CloudProvider,
		storagePrefix, stateKey string,
	) error

	// retrieves the storage instance name from the
	// storage backend configuration
	GetStorageInstanceName() string
}

// base cloud backend implementation
type cloudBackend struct {
	name,
	providerType string

	config interface{}
}

type newBackend func() (CloudBackend, error)

var backendNames = map[string]newBackend{
	"s3":      newS3Backend,
	"azurerm": newAzureRMBackend,
	"gcs":     newGCSBackend,
}

func NewCloudBackend(name string) (CloudBackend, error) {

	var (
		newBackend newBackend
		exists     bool
	)

	if newBackend, exists = backendNames[name]; !exists {
		return nil,
			fmt.Errorf("backend named '%s' is currently not handled", name)
	}
	return newBackend()
}

// out: a map of included cloud provider templates
func NewCloudBackendTemplates() (map[string]CloudBackend, error) {

	var (
		err error
	)

	templates := make(map[string]CloudBackend)
	for name, newBackend := range backendNames {
		if templates[name], err = newBackend(); err != nil {
			return nil, err
		}
	}
	return templates, nil
}

func IsValidCloudBackend(name string) bool {
	_, ok := backendNames[name]
	return ok
}

// interface: config/Configurable functions for base cloud provider

func (b *cloudBackend) Name() string {
	return b.name
}

func (b *cloudBackend) Description() string {
	return forms_config.CloudConfigForms.Group(b.name).Description()
}

func (b *cloudBackend) InputForm() (forms.InputForm, error) {

	var (
		err error
	)

	form := forms_config.CloudConfigForms.Group(b.name)
	if err = form.BindFields(b.config); err != nil {
		return nil, err
	}
	return form, nil
}

func (b *cloudBackend) GetValue(name string) (*string, error) {

	var (
		err error

		form  forms.InputForm
		field *forms.InputField
	)

	if form, err = b.InputForm(); err != nil {
		return nil, err
	}
	if field, err = form.GetInputField(name); err != nil {
		return nil, err
	}
	return field.Value(), nil
}

func (b *cloudBackend) Reset() {
}

// interface: backend/CloudBackend functions

func (b *cloudBackend) GetProviderType() string {
	return b.providerType
}

// interface: encoding/json/Unmarshaler

func (b *cloudBackend) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &b.config)
}

// interface: encoding/json/Marshaler

func (b *cloudBackend) MarshalJSON() ([]byte, error) {
	return json.Marshal(&b.config)
}
