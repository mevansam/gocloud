package backend

import (
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"

	forms_config "github.com/mevansam/gocloud/forms"
)

type s3Backend struct {
	cloudBackend

	isInitialized bool
}

type s3BackendConfig struct {
	Bucket *string `json:"bucket,omitempty" form_field:"bucket"`
	Key    *string `json:"key,omitempty" form_field:"key"`
}

func newS3Backend() (CloudBackend, error) {

	var (
		err           error
		beckendConfig s3BackendConfig
	)

	backend := &s3Backend{
		cloudBackend: cloudBackend{
			name:         "s3",
			providerType: "aws",

			config: &beckendConfig,
		},
		isInitialized: false,
	}
	err = backend.createS3InputForm()
	return backend, err
}

func (b *s3Backend) createS3InputForm() error {

	// Do not recreate form template if it exists
	clougConfig := forms_config.CloudConfigForms
	if clougConfig.HasGroup(b.name) {
		return nil
	}

	var (
		err  error
		form *forms.InputGroup
	)

	form = forms_config.CloudConfigForms.NewGroup(b.name, "Amazon Web Services S3 Storage Backend")

	if _, err = form.NewInputField(
		/* name */ "bucket",
		/* displayName */ "Bucket",
		/* description */ "The S3 bucket to store state in.",
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
		/* description */ "The key with which to identify the state object in the bucket.",
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

func (b *s3Backend) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudBackend
	)

	if copy, err = newS3Backend(); err != nil {
		return nil, err
	}

	config := b.cloudBackend.
		config.(*s3BackendConfig)
	configCopy := copy.(*s3Backend).cloudBackend.
		config.(*s3BackendConfig)

	*configCopy = *config

	return copy, nil
}

func (b *s3Backend) IsValid() bool {

	config := b.cloudBackend.
		config.(*s3BackendConfig)

	return config.Bucket != nil && len(*config.Bucket) > 0 &&
		config.Key != nil && len(*config.Key) > 0
}

// interface: backend/CloudBackend functions

func (b *s3Backend) Initialize(provider provider.CloudProvider) error {
	return nil
}