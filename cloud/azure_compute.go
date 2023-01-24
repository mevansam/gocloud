package cloud

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"

	"github.com/mevansam/goutils/logger"
)

type azureCompute struct {
	resourceGroupName,
	locationName,
	subscriptionID string

	ctx         context.Context	
	clientCreds *azidentity.ClientSecretCredential
	clientOpts  *arm.ClientOptions
}

type azureComputeInstance struct {
	id,
	name,

	publicIP,
	publicDNS,

	resourceGroupName,
	subscriptionID string

	ctx         context.Context	
	clientCreds *azidentity.ClientSecretCredential
	clientOpts  *arm.ClientOptions
}

func NewAzureCompute(
	ctx context.Context,
	clientCreds *azidentity.ClientSecretCredential,
	clientOpts *arm.ClientOptions,
	resourceGroupName,
	locationName,
	subscriptionID string,
) (Compute, error) {

	return &azureCompute{
		resourceGroupName: resourceGroupName,
		locationName:      locationName,

		subscriptionID: subscriptionID,

		ctx:         ctx,
		clientCreds: clientCreds,
		clientOpts:  clientOpts,
	}, nil
}

func (c *azureCompute) newAzureComputeInstance(
	resourceGroupName string,
	vm *armcompute.VirtualMachine,
) (*azureComputeInstance, error) {

	var (
		err error

		nicName string

		itfClient *armnetwork.InterfacesClient
		itf       armnetwork.InterfacesClientGetResponse

		addrClient *armnetwork.PublicIPAddressesClient
		addr       armnetwork.PublicIPAddressesClientGetResponse

		ipName,
		publicIP,
		publicFQDN string
	)

	logger.TraceMessage(
		"Creating Azure compute instance reference for VM '%s' in resource group '%s'.",
		*vm.Name, resourceGroupName,
	)

	networkProfile := vm.Properties.NetworkProfile

	if networkProfile.NetworkInterfaces != nil {
		nicItfList := networkProfile.NetworkInterfaces
		if len(nicItfList) == 1 {
			nicName = path.Base(*nicItfList[0].ID)
		} else {
			for _, nic := range nicItfList {
				if nic.Properties != nil {
					if nic.Properties.Primary != nil && *nic.Properties.Primary {
						nicName = path.Base(*nic.ID)
					}
				}
			}
		}
	}

	if len(nicName) > 0 {

		logger.TraceMessage(
			"Retrieving public IP of primary NIC '%s' of VM '%s' in resource group '%s'.",
			nicName, *vm.Name, resourceGroupName,
		)

		if itfClient, err = armnetwork.NewInterfacesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
			return nil, err
		}
		if itf, err = itfClient.Get(c.ctx, resourceGroupName, nicName, nil); err != nil {
			return nil, err
		}

		if itf.Properties.IPConfigurations != nil {
			ipConfigList := itf.Properties.IPConfigurations
			for _, ipConfig := range ipConfigList {
				if ipConfig.Properties.PublicIPAddress != nil {
					ipName = path.Base(*ipConfig.Properties.PublicIPAddress.ID)
				}
			}
		}

		if addrClient, err = armnetwork.NewPublicIPAddressesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
			return nil, err
		}
		if addr, err = addrClient.Get(c.ctx, resourceGroupName, ipName, nil); err != nil {
			return nil, err
		}

		if addr.Properties.IPAddress != nil {
			publicIP = *addr.Properties.IPAddress
		}
		if addr.Properties.DNSSettings != nil && addr.Properties.DNSSettings.Fqdn != nil {
			publicFQDN = *addr.Properties.DNSSettings.Fqdn
		}	
	}

	return &azureComputeInstance{
		id:   *vm.ID,
		name: *vm.Name,

		publicIP: publicIP,
		publicDNS: publicFQDN,

		resourceGroupName: resourceGroupName,
		subscriptionID:    c.subscriptionID,

		ctx:         c.ctx,
		clientCreds: c.clientCreds,
		clientOpts:  c.clientOpts,
	}, nil
}

// interface: cloud/Compute implementation

func (c *azureCompute) SetProperties(props interface{}) {
}

func (c *azureCompute) GetInstance(name string) (ComputeInstance, error) {

	var (
		err error

		client *armcompute.VirtualMachinesClient
		resp   armcompute.VirtualMachinesClientGetResponse
	)

	if client, err = armcompute.NewVirtualMachinesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
		return nil, err
	}
	if resp, err = client.Get(c.ctx, c.resourceGroupName, name, 
		&armcompute.VirtualMachinesClientGetOptions{
			Expand: to.Ptr(armcompute.InstanceViewTypesInstanceView),
		} ); err != nil {
		return nil, err
	}

	return c.newAzureComputeInstance(c.resourceGroupName, &resp.VirtualMachine)
}

func (c *azureCompute) GetInstances(ids []string) ([]ComputeInstance, error) {

	var (
		err error
		ok  bool

		elems, groupIDs   []string
		resourceGroupName string

		instances         []ComputeInstance
		filteredInstances []ComputeInstance
	)

	// map of resource groups to ids where resource
	// group name is extracted from the id path
	//
	// * /subscriptions
	//		/{subscriptionId}
	//			/resourceGroups
	//				/{resourceGroupName}
	//					/providers
	//						/Microsoft.Compute
	//							/virtualMachines
	//									/{vmName}
	//
	resourceGroups := make(map[string][]string)
	for _, id := range ids {
		elems = strings.Split(id, "/")
		if len(elems) != 9 {
			return nil,
				fmt.Errorf(
					"invalid id '%s'. it must have 8 elements", id,
				)
		}
		if elems[2] != c.subscriptionID {
			return nil,
				fmt.Errorf(
					"attempt retrieve an instance with subscription different from that configured",
				)
		}
		resourceGroupName = elems[4]
		if groupIDs, ok = resourceGroups[resourceGroupName]; ok {
			resourceGroups[resourceGroupName] = append(groupIDs, id)
		} else {
			resourceGroups[resourceGroupName] = []string{id}
		}
	}

	filteredInstances = make([]ComputeInstance, 0, len(ids))
	for resourceGroupName, groupIDs = range resourceGroups {
		// get all instances in resource
		// group and create filtered list
		if instances, err = c.listInstances(resourceGroupName); err != nil {
			return nil, err
		}
		for _, instance := range instances {
			for _, id := range groupIDs {
				if id == instance.ID() {
					filteredInstances = append(filteredInstances, instance)
					break
				}
			}
		}
	}
	return filteredInstances, nil
}

func (c *azureCompute) ListInstances() ([]ComputeInstance, error) {
	return c.listInstances(c.resourceGroupName)
}

func (c *azureCompute) listInstances(
	resourceGroupName string,
) ([]ComputeInstance, error) {

	var (
		err error

		client *armcompute.VirtualMachinesClient
		resp   armcompute.VirtualMachinesClientListResponse

		instance ComputeInstance
	)

	if client, err = armcompute.NewVirtualMachinesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
		return nil, err
	}

	listVMs := client.NewListPager(resourceGroupName, nil)
	instances := []ComputeInstance{}
	for listVMs.More() {
		if resp, err = listVMs.NextPage(c.ctx); err != nil {
			logger.ErrorMessage(
				"Failed to get next page of vm list for resource group '%s': %s", 
				resourceGroupName, err.Error(),
			)
			break
		}

		for _, vm := range resp.Value {
			if instance, err = c.newAzureComputeInstance(resourceGroupName, vm); err != nil {
				return nil, err
			}
			instances = append(instances, instance)
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

func (c *azureComputeInstance) PublicDNS() string {
	return c.publicDNS
}

func (c *azureComputeInstance) State() (InstanceState, error) {

	var (
		err error

		client *armcompute.VirtualMachinesClient
		resp   armcompute.VirtualMachinesClientInstanceViewResponse
	)

	if client, err = armcompute.NewVirtualMachinesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
		return StateUnknown, err
	}
	if resp, err = client.InstanceView(c.ctx, c.resourceGroupName, c.name, nil); err != nil {
		return StateUnknown, err
	}

	logger.TraceMessage("Status for azure VM '%s' in resource group '%s' is: %# v",
		c.name, c.resourceGroupName, resp.Statuses)

	if resp.Statuses != nil {
		statuses := resp.Statuses
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

		client *armcompute.VirtualMachinesClient
		presp  *runtime.Poller[armcompute.VirtualMachinesClientStartResponse]
	)

	if client, err = armcompute.NewVirtualMachinesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
		return err
	}

	logger.TraceMessage("Starting azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if presp, err = client.BeginStart(c.ctx, c.resourceGroupName, c.name, nil); err != nil {
		return err
	}
	if _, err = presp.PollUntilDone(c.ctx, nil); err != nil {
		return err
	}

	return err
}

func (c *azureComputeInstance) Restart() error {

	var (
		err error

		client *armcompute.VirtualMachinesClient
		presp  *runtime.Poller[armcompute.VirtualMachinesClientRestartResponse]
	)

	if client, err = armcompute.NewVirtualMachinesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
		return err
	}

	logger.TraceMessage("Restarting azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if presp, err = client.BeginRestart(c.ctx, c.resourceGroupName, c.name, nil); err != nil {
		return err
	}
	if _, err = presp.PollUntilDone(c.ctx, nil); err != nil {
		return err
	}

	return err
}

func (c *azureComputeInstance) Stop() error {

	var (
		err error

		client *armcompute.VirtualMachinesClient

		pOffResp     *runtime.Poller[armcompute.VirtualMachinesClientPowerOffResponse]
		pDeallocResp *runtime.Poller[armcompute.VirtualMachinesClientDeallocateResponse]
	)

	if client, err = armcompute.NewVirtualMachinesClient(c.subscriptionID, c.clientCreds, c.clientOpts); err != nil {
		return err
	}

	logger.TraceMessage("Powering off azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if pOffResp, err = client.BeginPowerOff(c.ctx, c.resourceGroupName, c.name, nil); err != nil {
		return err
	}
	if _, err = pOffResp.PollUntilDone(c.ctx, nil); err != nil {
		return err
	}

	logger.TraceMessage("Deallocating azure VM '%s' in resource group '%s'.",
		c.name, c.resourceGroupName)

	if pDeallocResp, err = client.BeginDeallocate(c.ctx, c.resourceGroupName, c.name, nil); err != nil {
		return err
	}
	if _, err = pDeallocResp.PollUntilDone(c.ctx, nil); err != nil {
		return err
	}
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
