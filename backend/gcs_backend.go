package backend

import (
	"fmt"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"

	forms_config "github.com/mevansam/gocloud/forms"
)

type gcsBackend struct {
	cloudBackend

	isInitialized bool
}

type gcsBackendConfig struct {
	Bucket *string `json:"bucket,omitempty" form_field:"bucket"`
	Prefix *string `json:"prefix,omitempty" form_field:"prefix"`
}

func newGCSBackend() (CloudBackend, error) {

	var (
		err           error
		beckendConfig gcsBackendConfig
	)

	backend := &gcsBackend{
		cloudBackend: cloudBackend{
			name:         "gcs",
			providerType: "google",

			config: &beckendConfig,
		},
		isInitialized: false,
	}
	err = backend.createGCSInputForm()
	return backend, err
}

func (p *gcsBackend) createGCSInputForm() error {

	// Do not recreate form template if it exists
	clougConfig := forms_config.CloudConfigForms
	if clougConfig.HasGroup(p.name) {
		return nil
	}

	var (
		err  error
		form *forms.InputGroup
	)

	form = forms_config.CloudConfigForms.NewGroup(p.name, "Google Cloud Storage Backend")

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "bucket",
		DisplayName: "Bucket",
		Description: "The GCS bucket to store state in.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target-undeployed"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "prefix",
		DisplayName: "Prefix",
		Description: "The prefix to use in the name of the state object.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target-undeployed"},
	}); err != nil {
		return err
	}

	return nil
}

// interface: config/Configurable functions of base cloud backend

func (b *gcsBackend) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudBackend
	)

	if copy, err = newGCSBackend(); err != nil {
		return nil, err
	}

	config := b.cloudBackend.
		config.(*gcsBackendConfig)
	configCopy := copy.(*gcsBackend).cloudBackend.
		config.(*gcsBackendConfig)

	configCopy.Bucket = utils.CopyStrPtr(config.Bucket)
	configCopy.Prefix = utils.CopyStrPtr(config.Prefix)

	return copy, nil
}

func (b *gcsBackend) IsValid() bool {

	config := b.cloudBackend.
		config.(*gcsBackendConfig)

	return config.Bucket != nil && len(*config.Bucket) > 0 &&
		config.Prefix != nil && len(*config.Prefix) > 0
}

// interface: backend/CloudBackend functions

func (b *gcsBackend) Configure(
	cloudProvider provider.CloudProvider,
	storagePrefix, stateKey string,
) error {

	var (
		err error

		inputForm forms.InputForm
		region    *string
	)

	if cloudProvider.Name() != b.providerType {
		return fmt.Errorf("the gcs backend can only be used with a google cloud provider")
	}
	if inputForm, err = cloudProvider.InputForm(); err != nil {
		return err
	}
	if region, err = inputForm.GetFieldValue("region"); err != nil {
		return err
	}
	if region == nil {
		return fmt.Errorf("google provider's region cannot be empty")
	}

	bucketName := fmt.Sprintf("%s-%s", storagePrefix, *region)

	config := b.cloudBackend.
		config.(*gcsBackendConfig)
	config.Bucket = &bucketName
	config.Prefix = &stateKey

	// rebind fields
	if _, err = b.InputForm(); err != nil {
		return err
	}
	logger.TraceMessage(
		"GCS backend configured using provider attributes: %# v",
		config)

	return nil
}
