package mocks

import (
	"github.com/mevansam/gocloud/provider"

	config_mocks "github.com/mevansam/goforms/test/mocks"
)

type FakeCloudBackend struct {
	config_mocks.FakeConfig
}

func NewFakeCloudBackend() *FakeCloudBackend {
	f := &FakeCloudBackend{}
	f.InitConfig("Test Cloud Backend Input", "Input form for mock cloud backend for testing")
	return f
}

func (f *FakeCloudBackend) Name() string {
	return "fake"
}

func (f *FakeCloudBackend) Description() string {
	return "fake cloud backend for testing"
}

func (f *FakeCloudBackend) GetProviderType() string {
	return "fake"
}

func (f *FakeCloudBackend) Configure(
	cloudProvider provider.CloudProvider,
	storagePrefix, stateKey string,
) error {
	return nil
}

func (f *FakeCloudBackend) GetStorageInstanceName() string {
	return "fake"
}
