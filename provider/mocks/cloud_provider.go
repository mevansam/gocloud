package mocks

import (
	"fmt"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"

	config_mocks "github.com/mevansam/goutils/config/mocks"
)

type FakeCloudProvider struct {
	config_mocks.FakeConfig
}

func NewFakeCloudProvider() *FakeCloudProvider {
	f := &FakeCloudProvider{}
	f.InitConfig("Test Cloud Provider Input", "Input form for mock cloud provider for testing")
	return f
}

func (f *FakeCloudProvider) Connect() error {
	return nil
}

func (f *FakeCloudProvider) Name() string {
	return "fake"
}

func (f *FakeCloudProvider) Description() string {
	return "fake cloud provider for testing"
}

func (f *FakeCloudProvider) Regions() []provider.RegionInfo {
	return []provider.RegionInfo{}
}

func (f *FakeCloudProvider) GetCompute() (cloud.Compute, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FakeCloudProvider) GetStorage() (cloud.Storage, error) {
	return nil, fmt.Errorf("not implemented")
}
