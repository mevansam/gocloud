package mocks

import (
	config_mocks "github.com/mevansam/goforms/config/mocks"
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
