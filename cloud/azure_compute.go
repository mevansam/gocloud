package cloud

import (
	"context"
	"fmt"
	"path"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/mevansam/goutils/logger"
)

type azureCompute struct {
	resourceGroupName,
	locationName,
	subscriptionID string

	ctx        context.Context
	authorizer *autorest.BearerAuthorizer
}

type azureComputeInstance struct {
	id,
	name,

	publicIP,

	resourceGroupName,
	subscriptionID string

	ctx        context.Context
	authorizer *autorest.BearerAuthorizer
}

func NewAzureCompute(
	ctx context.Context,
	authorizer *autorest.BearerAuthorizer,
	resourceGroupName,
	locationName,
	subscriptionID string,
) (Compute, error) {

	return &azureCompute{
		resourceGroupName: resourceGroupName,
		locationName:      locationName,

		subscriptionID: subscriptionID,

		ctx:        ctx,
		authorizer: authorizer,
	}, nil
}

func (c *azureCompute) newAzureComputeInstance(
	vm compute.VirtualMachine,
) (*azureComputeInstance, error) {

	var (
		err error

		nicName string
		nic     network.Interface

		ipName    string
		ipAddress network.PublicIPAddress

		publicIP string
	)

	logger.TraceMessage(
		"Creating Azure compute instance reference for VM '%s' in resource group '%s'.",
		*vm.Name, c.resourceGroupName,
	)

	if vm.NetworkProfile.NetworkInterfaces != nil {
		nicItfList := *vm.NetworkProfile.NetworkInterfaces
		if len(nicItfList) == 1 {
			nicName = path.Base(*nicItfList[0].ID)
		} else {
			for _, nic := range nicItfList {
				if nic.NetworkInterfaceReferenceProperties != nil {
					if nic.Primary != nil && *nic.Primary {
						nicName = path.Base(*nic.ID)
					}
				}
			}
		}
	}

	if len(nicName) > 0 {
		nicClient := network.NewInterfacesClient(c.subscriptionID)
		nicClient.Authorizer = c.authorizer
		nicClient.AddToUserAgent(httpUserAgent)

		logger.TraceMessage(
			"Retrieving public IP of primary NIC '%s' of VM '%s' in resource group '%s'.",
			nicName, *vm.Name, c.resourceGroupName,
		)

		if nic, err = nicClient.Get(c.ctx,
			c.resourceGroupName,
			nicName, ""); err != nil {
			return nil, err
		}

		if nic.IPConfigurations != nil {
			ipConfigList := *nic.IPConfigurations
			for _, ipConfig := range ipConfigList {
				if ipConfig.InterfaceIPConfigurationPropertiesFormat != nil &&
					(*ipConfig.InterfaceIPConfigurationPropertiesFormat).PublicIPAddress != nil {

					ipName = path.Base(*(*(*ipConfig.InterfaceIPConfigurationPropertiesFormat).PublicIPAddress).ID)
				}
			}
		}

		addressClient := network.NewPublicIPAddressesClient(c.subscriptionID)
		addressClient.Authorizer = c.authorizer
		addressClient.AddToUserAgent(httpUserAgent)

		if ipAddress, err = addressClient.Get(c.ctx,
			c.resourceGroupName,
			ipName,
			"",
		); err != nil {
			return nil, err
		}
		if ipAddress.PublicIPAddressPropertiesFormat.IPAddress != nil {
			publicIP = *ipAddress.PublicIPAddressPropertiesFormat.IPAddress
		}
	}

	return &azureComputeInstance{
		id:   *vm.ID,
		name: *vm.Name,

		publicIP: publicIP,

		resourceGroupName: c.resourceGroupName,
		subscriptionID:    c.subscriptionID,

		ctx:        c.ctx,
		authorizer: c.authorizer,
	}, nil
}

// interface: cloud/Compute implementation

func (c *azureCompute) SetProperties(props interface{}) {
}

func (c *azureCompute) GetInstance(name string) (ComputeInstance, error) {

	var (
		err error

		vm compute.VirtualMachine
	)

	vmClient := compute.NewVirtualMachinesClient(c.subscriptionID)
	vmClient.Authorizer = c.authorizer
	vmClient.AddToUserAgent(httpUserAgent)

	if vm, err = vmClient.Get(c.ctx,
		c.resourceGroupName,
		name,
		compute.InstanceView,
	); err != nil {
		return nil, err
	}

	return c.newAzureComputeInstance(vm)
}

func (c *azureCompute) GetInstances(ids []string) ([]ComputeInstance, error) {

	var (
		err error

		instances         []ComputeInstance
		filteredInstances []ComputeInstance
	)

	// get all instances in resource
	// group and create filtered list
	if instances, err = c.ListInstances(); err != nil {
		return nil, err
	}
	filteredInstances = make([]ComputeInstance, 0, len(instances))

	for _, instance := range instances {
		for _, id := range ids {
			if id == instance.ID() {
				filteredInstances = append(filteredInstances, instance)
				break
			}
		}
	}
	return filteredInstances, nil
}

func (c *azureCompute) ListInstances() ([]ComputeInstance, error) {

	var (
		err error

		items compute.VirtualMachineListResultIterator
		value compute.VirtualMachine

		instance ComputeInstance
	)

	vmClient := compute.NewVirtualMachinesClient(c.subscriptionID)
	vmClient.Authorizer = c.authorizer
	vmClient.AddToUserAgent(httpUserAgent)

	if items, err = vmClient.ListComplete(c.ctx, c.resourceGroupName); err != nil {
		return nil, err
	}

	instances := []ComputeInstance{}
	for items.NotDone() {
		value = items.Value()

		if instance, err = c.newAzureComputeInstance(value); err != nil {
			return nil, err
		}
		instances = append(instances, instance)

		if err = items.Next(); err != nil {
			return nil, err
		}
	}

	return instances, nil
}

// interface: cloud/ComputeInstance implementation

func (c *azureComputeInstance) ID() string {
	return c.id
}

func (c *azureComputeInstance) Name() string {
	return c.name
}

func (c *azureComputeInstance) PublicIP() string {
	return c.publicIP
}

func (c *azureComputeInstance) State() (InstanceState, error) {

	var (
		err error

		instanceView compute.VirtualMachineInstanceView
	)

	vmClient := compute.NewVirtualMachinesClient(c.subscriptionID)
	vmClient.Authorizer = c.authorizer
	vmClient.AddToUserAgent(httpUserAgent)

	if instanceView, err = vmClient.InstanceView(c.ctx,
		c.resourceGroupName,
		c.name,
	); err != nil {
		return StateUnknown, err
	}
	logger.TraceMessage("Status for azure VM '%s' in resource group '%s' is: %# v",
		c.name, c.resourceGroupName, instanceView.Statuses)

	if instanceView.Statuses != nil {
		statuses := *instanceView.Statuses
		if len(statuses) > 1 {
			status := statuses[len(statuses)-1].DisplayStatus
			if status != nil {
				switch *status {
				case "VM running":
					return StateRunning, nil
				case "VM stopped", "VM deallocated":
					return StateStopped, nil
				default:
					return StatePending, nil
				}
			}
		}
	}
	return StateUnknown, nil
}

func (c *azureComputeInstance) Start() error {

	var (
		err error

		startFuture compute.VirtualMachinesStartFuture
	)

	vmClient := compute.NewVirtualMachinesClient(c.subscriptionID)
	vmClient.Authorizer = c.authorizer
	vmClient.AddToUserAgent(httpUserAgent)

	logger.TraceMessage("Starting azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if startFuture, err = vmClient.Start(c.ctx,
		c.resourceGroupName,
		c.name,
	); err != nil {
		return err
	}
	err = startFuture.WaitForCompletionRef(c.ctx, vmClient.BaseClient.Client)
	return err
}

func (c *azureComputeInstance) Restart() error {

	var (
		err error

		restartFuture compute.VirtualMachinesRestartFuture
	)

	vmClient := compute.NewVirtualMachinesClient(c.subscriptionID)
	vmClient.Authorizer = c.authorizer
	vmClient.AddToUserAgent(httpUserAgent)

	logger.TraceMessage("Restarting azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if restartFuture, err = vmClient.Restart(c.ctx,
		c.resourceGroupName,
		c.name,
	); err != nil {
		return err
	}
	err = restartFuture.WaitForCompletionRef(c.ctx, vmClient.BaseClient.Client)
	return err
}

func (c *azureComputeInstance) Stop() error {

	var (
		err error

		offFuture     compute.VirtualMachinesPowerOffFuture
		deallocFuture compute.VirtualMachinesDeallocateFuture
	)

	vmClient := compute.NewVirtualMachinesClient(c.subscriptionID)
	vmClient.Authorizer = c.authorizer
	vmClient.AddToUserAgent(httpUserAgent)

	logger.TraceMessage("Powering off azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if offFuture, err = vmClient.PowerOff(c.ctx,
		c.resourceGroupName,
		c.name,
		nil,
	); err != nil {
		return err
	}
	if err = offFuture.WaitForCompletionRef(c.ctx, vmClient.BaseClient.Client); err != nil {
		return err
	}

	logger.TraceMessage("Deallocating azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if deallocFuture, err = vmClient.Deallocate(c.ctx,
		c.resourceGroupName,
		c.name,
	); err != nil {
		return err
	}
	err = deallocFuture.WaitForCompletionRef(c.ctx, vmClient.BaseClient.Client)
	return err
}

func (c *azureComputeInstance) CanConnect(port int) bool {

	if c.publicIP != "" {
		return canConnect(
			fmt.Sprintf("%s:%d", c.PublicIP(), port),
		)
	} else {
		return false
	}
}
