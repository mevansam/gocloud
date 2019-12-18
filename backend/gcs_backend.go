package backend

import (
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"

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
		Tags:        []string{"backend", "target"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "prefix",
		DisplayName: "Prefix",
		Description: "The prefix to use in the name of the state object.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target"},
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

	*configCopy = *config

	return copy, nil
}

func (b *gcsBackend) IsValid() bool {

	config := b.cloudBackend.
		config.(*gcsBackendConfig)

	return config.Bucket != nil && len(*config.Bucket) > 0 &&
		config.Prefix != nil && len(*config.Prefix) > 0
}

// interface: backend/CloudBackend functions

func (b *gcsBackend) Initialize(provider provider.CloudProvider) error {
	return nil
}
