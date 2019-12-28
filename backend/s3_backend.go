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

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "bucket",
		DisplayName: "Bucket",
		Description: "The S3 bucket to store state in.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target-undeployed"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "key",
		DisplayName: "Key",
		Description: "The key with which to identify the state object in the bucket.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target-undeployed"},
	}); err != nil {
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

	configCopy.Bucket = utils.CopyStrPtr(config.Bucket)
	configCopy.Key = utils.CopyStrPtr(config.Key)

	return copy, nil
}

func (b *s3Backend) IsValid() bool {

	config := b.cloudBackend.
		config.(*s3BackendConfig)

	return config.Bucket != nil && len(*config.Bucket) > 0 &&
		config.Key != nil && len(*config.Key) > 0
}

// interface: backend/CloudBackend functions

func (b *s3Backend) Configure(
	cloudProvider provider.CloudProvider,
	storagePrefix, stateKey string,
) error {

	var (
		err error

		inputForm forms.InputForm
		region    *string
	)

	if cloudProvider.Name() != b.providerType {
		return fmt.Errorf("the s3 backend can only be used with an aws cloud provider")
	}
	if inputForm, err = cloudProvider.InputForm(); err != nil {
		return err
	}
	if region, err = inputForm.GetFieldValue("region"); err != nil {
		return err
	}
	if region == nil {
		return fmt.Errorf("aws provider's region cannot be empty")
	}

	bucketName := fmt.Sprintf("%s-%s", storagePrefix, *region)

	config := b.cloudBackend.
		config.(*s3BackendConfig)
	config.Bucket = &bucketName
	config.Key = &stateKey

	// rebind fields
	if _, err = b.InputForm(); err != nil {
		return err
	}
	logger.TraceMessage(
		"S3 backend configured using provider attributes: %# v",
		config)

	return nil
}

func (b *s3Backend) GetStorageInstanceName() string {
	return *b.cloudBackend.config.(*s3BackendConfig).Bucket
}
