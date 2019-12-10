package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/google/uuid"
	"github.com/mevansam/gocloud/provider"

	"github.com/mevansam/goutils/forms"
	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (

	// azure credentials from environment
	azureClientID,
	azureClientSecret,
	azureTenantID,
	azureSubscriptionID,
	azureDefaultResourceGroup string
)

var azureEnvAuthMap = make(map[string]*autorest.BearerAuthorizer)
var azureEnvLocMap = make(map[string]map[string]string)

// read azure environment
func InitializeAzureEnvironment() {

	defer GinkgoRecover()

	// retrieve azure credentials from environment
	if azureClientID = os.Getenv("ARM_CLIENT_ID"); len(azureClientID) == 0 {
		Fail("environment variable named ARM_CLIENT_ID must be provided")
	}
	if azureClientSecret = os.Getenv("ARM_CLIENT_SECRET"); len(azureClientSecret) == 0 {
		Fail("environment variable named ARM_CLIENT_SECRET must be provided")
	}
	if azureTenantID = os.Getenv("ARM_TENANT_ID"); len(azureTenantID) == 0 {
		Fail("environment variable named ARM_TENANT_ID must be provided")
	}
	if azureSubscriptionID = os.Getenv("ARM_SUBSCRIPTION_ID"); len(azureSubscriptionID) == 0 {
		Fail("environment variable named ARM_SUBSCRIPTION_ID must be provided")
	}

	if noCleanUp := os.Getenv("ARM_NO_CLEANUP"); noCleanUp == "1" {
		azureDefaultResourceGroup = "cb_default_test"
	} else {
		azureDefaultResourceGroup = fmt.Sprintf("cb_default_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	}
}

// update azure provider with environment credentials
func InitializeAzureProvider(azureProvider provider.CloudProvider) {

	var (
		err error

		inputForm forms.InputForm
	)

	inputForm, err = azureProvider.InputForm()
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("subscription_id", azureSubscriptionID)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("client_id", azureClientID)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("client_secret", azureClientSecret)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("tenant_id", azureTenantID)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("default_resource_group", azureDefaultResourceGroup)
	Expect(err).NotTo(HaveOccurred())
}

// cleans up any test data created in Azure account
func CleanUpAzureTestData() {

	if noCleanUp := os.Getenv("ARM_NO_CLEANUP"); noCleanUp != "1" {
		DeleteAzureResourceGroup(azureDefaultResourceGroup)
	}
}

func DeleteAzureResourceGroup(azureResourceGroup string) {

	var (
		err error
	)

	// clean up azure cloud account
	client := AzureGroupsClient("AzurePublicCloud")
	ctx := context.Background()

	if _, err = client.Get(ctx, azureResourceGroup); err == nil {
		_, err = client.Delete(ctx, azureResourceGroup)
		Expect(err).NotTo(HaveOccurred())
	}
}

// initialzie Azure authorizer for a particular region.
func azureAuthorizer(environment string) *autorest.BearerAuthorizer {

	var (
		err error

		authorizer *autorest.BearerAuthorizer
		exists     bool
	)

	if authorizer, exists = azureEnvAuthMap[environment]; !exists {

		var (
			env         azure.Environment
			oauthConfig *adal.OAuthConfig
			token       *adal.ServicePrincipalToken
		)

		if env, err = azure.EnvironmentFromName(environment); err != nil {
			Fail(err.Error())
		}
		if oauthConfig, err = adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, azureTenantID); err != nil {
			Fail(err.Error())
		}
		if token, err = adal.NewServicePrincipalToken(*oauthConfig, azureClientID, azureClientSecret, env.ResourceManagerEndpoint); err != nil {
			Fail(err.Error())
		}
		authorizer = autorest.NewBearerAuthorizer(token)
		azureEnvAuthMap[environment] = authorizer
	}
	return authorizer
}

// return azure groups client
func AzureGroupsClient(environment string) resources.GroupsClient {
	client := resources.NewGroupsClient(azureSubscriptionID)
	client.Authorizer = azureAuthorizer("AzurePublicCloud")
	client.AddToUserAgent("cbs-test")
	return client
}

// load azure regions/locations via API calls
func AzureLocations(environment string) map[string]string {

	var (
		err error

		locationMap map[string]string
		exists      bool
	)

	if locationMap, exists = azureEnvLocMap[environment]; !exists {
		var (
			client subscriptions.Client
			result subscriptions.LocationListResult
		)

		client = subscriptions.NewClient()
		client.Authorizer = azureAuthorizer(environment)
		client.AddToUserAgent("cbs-test")

		result, err = client.ListLocations(context.Background(), azureSubscriptionID)
		if err != nil {
			Fail(err.Error())
		}

		logger.TraceMessage("\nAzure locations retrieved from API call:")
		locationMap = make(map[string]string)
		for _, l := range *result.Value {
			logger.TraceMessage("  * %s - %s", *l.Name, *l.DisplayName)
			locationMap[*l.Name] = *l.DisplayName
		}
		azureEnvLocMap[environment] = locationMap
	}
	return locationMap
}

// creates azure instances for testing
func AzureDeployTestInstances(name string, numInstances int) map[string]string {

	var (
		err error

		instanceIPs map[string]string
		ipAddress   network.PublicIPAddress

		future resources.DeploymentsCreateOrUpdateFuture
	)

	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath := path.Dir(filename)
	templateFile := sourceDirPath + "/azure_vm_template.json"
	templateFileData, err := ioutil.ReadFile(templateFile)
	Expect(err).NotTo(HaveOccurred())

	template := make(map[string]interface{})
	err = json.Unmarshal(templateFileData, &template)
	Expect(err).NotTo(HaveOccurred())

	vmClient := compute.NewVirtualMachinesClient(azureSubscriptionID)
	vmClient.Authorizer = azureAuthorizer("AzurePublicCloud")
	vmClient.AddToUserAgent("cbs-test")

	deploymentsClient := resources.NewDeploymentsClient(azureSubscriptionID)
	deploymentsClient.Authorizer = azureAuthorizer("AzurePublicCloud")
	deploymentsClient.AddToUserAgent("cbs-test")

	addressClient := network.NewPublicIPAddressesClient(azureSubscriptionID)
	addressClient.Authorizer = azureAuthorizer("AzurePublicCloud")
	addressClient.AddToUserAgent("cbs-test")

	instanceIPs = make(map[string]string)

	ctx := context.Background()
	for i := 0; i < numInstances; i++ {

		vmName := fmt.Sprintf("%s-%d", name, i)
		ipName := fmt.Sprintf("cbstestip-%s-%d", name, i)

		if _, err = vmClient.Get(ctx, azureDefaultResourceGroup, vmName, compute.InstanceView); err != nil {
			// create test vm if it has not been created
			logger.TraceMessage("Deploying test VM '%s'.", vmName)

			future, err = deploymentsClient.CreateOrUpdate(ctx,
				azureDefaultResourceGroup,
				fmt.Sprintf("cbsAzureTestVM-%s-%d", name, i),
				resources.Deployment{
					Properties: &resources.DeploymentProperties{
						Template: template,
						Parameters: map[string]interface{}{
							"virtualMachines_name": map[string]interface{}{
								"value": vmName,
							},
							"networkInterfaces_name": map[string]interface{}{
								"value": fmt.Sprintf("cbstestnic-%s-%d", name, i),
							},
							"publicIPAddresses_name": map[string]interface{}{
								"value": ipName,
							},
						},
						Mode: resources.Incremental,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			err = future.WaitForCompletionRef(ctx, deploymentsClient.BaseClient.Client)
			Expect(err).NotTo(HaveOccurred())
		}

		ipAddress, err = addressClient.Get(ctx,
			azureDefaultResourceGroup,
			ipName,
			"",
		)
		Expect(err).NotTo(HaveOccurred())
		if ipAddress.PublicIPAddressPropertiesFormat.IPAddress != nil {
			instanceIPs[vmName] = *ipAddress.PublicIPAddressPropertiesFormat.IPAddress
		} else {
			instanceIPs[vmName] = ""
		}
		logger.TraceMessage(
			"IP address for VM '%s' is '%s'.",
			vmName, instanceIPs[vmName])
	}

	return instanceIPs
}

func AzureInstanceState(name string) string {

	var (
		err error

		instanceView compute.VirtualMachineInstanceView
	)

	vmClient := compute.NewVirtualMachinesClient(azureSubscriptionID)
	vmClient.Authorizer = azureAuthorizer("AzurePublicCloud")
	vmClient.AddToUserAgent("cbs-test")

	instanceView, err = vmClient.InstanceView(context.Background(),
		azureDefaultResourceGroup,
		name,
	)
	Expect(err).NotTo(HaveOccurred())

	if instanceView.Statuses != nil {
		statuses := *instanceView.Statuses
		if len(statuses) > 1 {
			return *statuses[len(statuses)-1].DisplayStatus
		}
	}
	return ""
}

// returns if the given storage account exists
func AzureStorageAccountExists(name, resourceGroup string) (bool, error) {

	var (
		err error

		storageAccts storage.AccountListResult
		exists       bool
	)

	// check if default storage account exists
	saClient := storage.NewAccountsClient(azureSubscriptionID)
	saClient.Authorizer = azureAuthorizer("AzurePublicCloud")
	saClient.AddToUserAgent("cbs-test")

	if storageAccts, err = saClient.ListByResourceGroup(
		context.Background(),
		resourceGroup,
	); err != nil {
		return false, err
	}

	exists = false
	for _, acct := range *storageAccts.Value {
		if *acct.Name == name {
			exists = true
			break
		}
	}
	if !exists {
		return false, nil
	}

	return true, nil
}
