package backend

import (
	"fmt"
	"os"
	"path/filepath"

	forms_config "github.com/mevansam/gocloud/forms"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)


type localBackend struct {
	cloudBackend

	statePath string
}

type localBackendConfig struct {
	Path *string `json:"path,omitempty" form_field:"path"`
}

func newLocalBackend() (CloudBackend, error) {

	var (
		err           error
		beckendConfig localBackendConfig
	)

	backend := &localBackend{
		cloudBackend: cloudBackend{
			name:   "local",
			config: &beckendConfig,
		},
	}
	err = backend.createLocalInputForm()
	return backend, err
}

func (b *localBackend) createLocalInputForm() error {

	// Do not recreate form template if it exists
	clougConfig := forms_config.CloudConfigForms
	if clougConfig.HasGroup(b.name) {
		return nil
	}

	var (
		err  error
		form *forms.InputGroup
	)

	form = forms_config.CloudConfigForms.NewGroup(b.name, "Local Storage Backend")

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "path",
		DisplayName: "Path",
		Description: "Local path to save state in.",
		InputType:   forms.String,
		Tags:        []string{"backend", "target-undeployed"},

		InclusionFilter:             `^([a-zA-Z]:)?[\\\/](([^\\\/:*?"<>|]+)[\\\/]?)*$`,
		InclusionFilterErrorMessage: "Local file system path has an an invalid format.",
	}); err != nil {
		return err
	}	
	return nil
}

// interface: config/Configurable functions of base cloud backend

func (b *localBackend) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudBackend
	)

	if copy, err = newLocalBackend(); err != nil {
		return nil, err
	}
	copy.(*localBackend).statePath = b.statePath

	config := b.cloudBackend.
		config.(*localBackendConfig)
	configCopy := copy.(*localBackend).cloudBackend.
		config.(*localBackendConfig)

	configCopy.Path = utils.CopyStrPtr(config.Path)
	return copy, nil
}

func (b *localBackend) IsValid() bool {

	config := b.cloudBackend.
		config.(*localBackendConfig)

	return config.Path != nil && len(*config.Path) > 0
}

// interface: backend/CloudBackend functions

func (b *localBackend) SetProperties(props interface{}) {
	b.statePath = props.(CloudBackendProperties).StatePath
}

func (b *localBackend) Configure(
	cloudProvider provider.CloudProvider,
	storagePrefix, stateKey string,
) error {

	info, err := os.Stat(b.statePath)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(b.statePath, 0755); err != nil {
			return err
		}
	} else {
		if !info.IsDir() {
			return fmt.Errorf("local backend state path '%s' is not valid", b.statePath)
		}	
	}

	localStatePath := filepath.Join(b.statePath, storagePrefix, "local.tfstate")
	config := b.cloudBackend.
		config.(*localBackendConfig)
	config.Path = &localStatePath
	
	// rebind fields
	if _, err = b.InputForm(); err != nil {
		return err
	}
	logger.TraceMessage(
		"Local backend configured using provider attributes: %# v",
		config)

	return nil
}

func (b *localBackend) GetStorageInstanceName() string {

	config := b.cloudBackend.
		config.(*localBackendConfig)

	return *config.Path
}
