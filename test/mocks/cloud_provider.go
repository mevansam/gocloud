package mocks

import (
	"fmt"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goutils/utils"

	config_mocks "github.com/mevansam/goforms/test/mocks"
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

func (f *FakeCloudProvider) Region() *string {
	return utils.PtrToStr("us-east-1")
}

func (f *FakeCloudProvider) GetRegions() []provider.RegionInfo {
	return []provider.RegionInfo{}
}

func (f *FakeCloudProvider) GetCompute() (cloud.Compute, error) {

	testInstance := FakeComputeInstance{
		id: "bastion-instance-id",
		name: "bastion",
	}

	return &FakeCompute{
		Instances: []cloud.ComputeInstance{&testInstance},
	}, nil
}

func (f *FakeCloudProvider) GetStorage() (cloud.Storage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FakeCloudProvider) GetVars(vars map[string]string) error {
	return fmt.Errorf("not implemented")
}

type FakeCompute struct {
	Instances []cloud.ComputeInstance
}

func (f *FakeCompute) SetProperties(props interface{}) {	
}

func (f *FakeCompute) GetInstance(name string) (cloud.ComputeInstance, error) {
	for _, i := range f.Instances {
		if name == i.Name() {
			return i, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (f *FakeCompute) GetInstances(ids []string) ([]cloud.ComputeInstance, error) {
	instances := []cloud.ComputeInstance{}
	for _, i := range f.Instances {
		for _, id := range ids {
			if id == i.ID() {
				instances = append(instances, i)
			}	
		}
	}
	return instances, nil
}

func (f *FakeCompute) ListInstances() ([]cloud.ComputeInstance, error) {
	return f.Instances, nil
}

type FakeComputeInstance struct {
	id,
	name,
	publicIP string

	state cloud.InstanceState
}

func (i *FakeComputeInstance) SetValues(
	id,
	name,
	publicIP string,
	state cloud.InstanceState,
) {
	i.id = id
	i.name = name
	i.publicIP = publicIP
	i.state = state
}

func (i *FakeComputeInstance) ID() string {
	return i.id
}

func (i *FakeComputeInstance) Name() string {
	return i.name
}

func (i *FakeComputeInstance) PublicIP() string {
	return i.publicIP
}

func (i *FakeComputeInstance) PublicDNS() string {
	return ""
}

func (i *FakeComputeInstance) State() (cloud.InstanceState, error) {
	return i.state, nil
}

func (i *FakeComputeInstance) Start() error {
	return fmt.Errorf("not implemented")
}

func (i *FakeComputeInstance) Restart() error {
	return fmt.Errorf("not implemented")
}

func (i *FakeComputeInstance) Stop() error {
	return fmt.Errorf("not implemented")
}

func (i *FakeComputeInstance) CanConnect(port int) bool {
	return false
}
