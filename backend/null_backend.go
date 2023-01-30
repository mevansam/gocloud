package backend

import (
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
)


type nullBackend struct {
	cloudBackend
}

type nullBackendConfig struct {
}

func newNullBackend(name string) (CloudBackend, error) {

	var (
		err           error
		beckendConfig nullBackendConfig
	)

	backend := &nullBackend{
		cloudBackend: cloudBackend{
			name:   name,
			config: &beckendConfig,
		},
	}
	return backend, err
}

// interface: config/Configurable functions of base cloud backend

func (p *nullBackend) InputForm() (forms.InputForm, error) {
	return nil, nil
}

func (b *nullBackend) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudBackend
	)

	if copy, err = newLocalBackend(); err != nil {
		return nil, err
	}
	return copy, nil
}

func (b *nullBackend) IsValid() bool {
	return true
}

// interface: backend/CloudBackend functions

func (b *nullBackend) Configure(
	cloudProvider provider.CloudProvider,
	storagePrefix, stateKey string,
) error {
	return nil
}

func (b *nullBackend) GetStorageInstanceName() string {
	return ""
}

func (b *nullBackend) GetVars(vars map[string]string) error {
	return nil
}