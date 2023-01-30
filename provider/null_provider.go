package provider

import (
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
)

type nullProvider struct {
	cloudProvider
}

type nullProviderConfig struct {
}

func newNullProvider(name string) (CloudProvider, error) {

	return &nullProvider{
		cloudProvider: cloudProvider{
			name:   name,
			config: &nullProviderConfig{},
		},
	}, nil
}

// interface: config/Configurable functions of base cloud provider

func (p *nullProvider) InputForm() (forms.InputForm, error) {
	return nil, nil
}

func (p *nullProvider) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudProvider
	)

	if copy, err = newNullProvider(p.name); err != nil {
		return nil, err
	}
	return copy, nil
}

func (p *nullProvider) IsValid() bool {
	return true
}

// interface: config/provider/CloudProvider functions

func (p *nullProvider) Connect() error {
	return nil
}

func (p *nullProvider) Region() *string {
	return nil
}

func (p *nullProvider) GetRegions() []RegionInfo {
	return []RegionInfo{}
}

func (p *nullProvider) GetCompute() (cloud.Compute, error) {
	return nil, nil
}

func (p *nullProvider) GetStorage() (cloud.Storage, error) {
	return nil, nil
}

func (p *nullProvider) GetVars(vars map[string]string) error {
	return nil
}