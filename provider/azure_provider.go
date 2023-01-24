package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mevansam/goutils/rest"
	"github.com/mevansam/goutils/utils"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/google/uuid"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	forms_config "github.com/mevansam/gocloud/forms"
)

/*
Setting up AZ Account:

1) Go to https://portal.azure.com/#view/Microsoft_Azure_Billing/SubscriptionsBlade and add a new subscription
2) Install the Azure CLI - https://learn.microsoft.com/en-us/cli/azure/install-azure-cli.
3) From a command/terminal window run:
az logout && az login
az account set --subscription="SUBSCRIPTION_ID"
az ad sp create-for-rbac --role="Owner" --scopes="/subscriptions/SUBSCRIPTION_ID"

*/

type azureProvider struct {
	cloudProvider

	// background context
	ctx context.Context

	// indicates if provider client has
	// been prepared to make API requests
	isInitialized bool

	clientCreds   *azidentity.ClientSecretCredential
	clientOpts    *arm.ClientOptions
	defaultResGrp armresources.ResourceGroup

	servicePrincipalID string
}

type azureProviderConfig struct {
	Environment          *string `json:"environment,omitempty" form_field:"environment"`
	SubscriptionID       *string `json:"subscription_id,omitempty" form_field:"subscription_id"`
	ClientID             *string `json:"client_id,omitempty" form_field:"client_id"`
	ClientSecret         *string `json:"client_secret,omitempty" form_field:"client_secret"`
	TenantID             *string `json:"tenant_id,omitempty" form_field:"tenant_id"`
	DefaultResourceGroup *string `json:"default_resource_group,omitempty" form_field:"default_resource_group"`
	DefaultLocation      *string `json:"default_location,omitempty" form_field:"default_location"`
}

var environments = map[string]azcloud.Configuration{
	"public":       azcloud.AzurePublic,
	"usgovernment": azcloud.AzureGovernment,
	"china":        azcloud.AzureChina,
}

var saNameRegex = regexp.MustCompile(`[-_:]`)

func newAzureProvider() (CloudProvider, error) {

	var (
		err            error
		providerConfig azureProviderConfig
	)

	provider := &azureProvider{
		cloudProvider: cloudProvider{
			name:   "azure",
			config: &providerConfig,
		},
		ctx:           context.Background(),
		isInitialized: false,
	}
	err = provider.createAzureInputForm()
	return provider, err
}

func (p *azureProvider) createAzureInputForm() error {

	// Do not recreate form template if it exists
	clougConfig := forms_config.CloudConfigForms
	if clougConfig.HasGroup(p.name) {
		return nil
	}

	var (
		err  error
		form *forms.InputGroup
	)

	regions := p.GetRegions()
	regionList := make([]string, len(regions))
	for i, r := range regions {
		regionList[i] = r.Name
	}

	envList := make([]string, 0, len(environments))
	for k := range environments {
		envList = append(envList, k)
	}

	form = forms_config.CloudConfigForms.NewGroup(p.name, "Microsoft Azure Cloud Computing Platform")

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:         "environment",
		DisplayName:  "Environment",
		Description:  "The Azure environment.",
		InputType:    forms.String,
		DefaultValue: utils.PtrToStr("public"),
		EnvVars: []string{
			"ARM_ENVIRONMENT",
		},
		Tags:                       []string{"provider"},
		AcceptedValues:             envList,
		AcceptedValuesErrorMessage: "Not a valid Azure environment.",
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "subscription_id",
		DisplayName: "Subscription ID",
		Description: "The Azure subscription ID.",
		InputType:   forms.String,
		EnvVars: []string{
			"ARM_SUBSCRIPTION_ID",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "client_id",
		DisplayName: "Client ID",
		Description: "The Client ID or Application ID of the Azure Service Principal.",
		InputType:   forms.String,
		EnvVars: []string{
			"ARM_CLIENT_ID",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "client_secret",
		DisplayName: "Client Secret",
		Description: "The Client Secret or Password of the Azure Service Principal.",
		InputType:   forms.String,
		Sensitive:   true,
		EnvVars: []string{
			"ARM_CLIENT_SECRET",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "tenant_id",
		DisplayName: "Tenant ID",
		Description: "The Tenant ID from the Azure Service Principal.",
		InputType:   forms.String,
		EnvVars: []string{
			"ARM_TENANT_ID",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}

	// Default resource group and location

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "default_resource_group",
		DisplayName: "Default Resource Group",
		Description: "Resource group where common resources will be created.",
		InputType:   forms.String,
		DefaultValue: utils.PtrToStr(fmt.Sprintf(
			"cb_default_%s",
			strings.ReplaceAll(uuid.New().String(), "-", ""),
		)),
		Tags: []string{"provider", "target-undeployed"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:         "default_location",
		DisplayName:  "Default Location or Region",
		Description:  "The location of the default resource group.",
		InputType:    forms.String,
		DefaultValue: utils.PtrToStr("eastus"),
		Tags:         []string{"provider", "target-undeployed"},

		AcceptedValues:             regionList,
		AcceptedValuesErrorMessage: "Not a valid AWS region.",
	}); err != nil {
		return err
	}

	return nil
}

func (p *azureProvider) connect(config *azureProviderConfig) error {

	var (
		err error
		ok  bool

		env     azcloud.Configuration
		options azidentity.ClientSecretCredentialOptions

		token azcore.AccessToken
	)

	if env, ok = environments[*config.Environment]; !ok {
		env = azcloud.AzurePublic
	}
	options.ClientOptions = azcore.ClientOptions{
		Cloud: env,
	}
	p.clientOpts = &arm.ClientOptions{
		ClientOptions: options.ClientOptions,
	}

	if p.clientCreds, err = azidentity.NewClientSecretCredential(
		*config.TenantID, 
		*config.ClientID, 
		*config.ClientSecret, 
		&options,
	); err != nil {
		return err
	}

	// To assign roles/permissions the principal/object id of
	// the service principal (i.e. client_id) used is required.
	// There is no azure sdk function to retrieve this so a
	// direct rest api call is made to the Microsoft Graph API.
	// Once azure SDK implements a suitable function to the
	// following should be refactored.
	//
	// Equivalent CLI cmd:
	//
	// az ad sp show --id $ARM_CLIENT_ID

	if token, err = p.clientCreds.GetToken(p.ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com//.default"},
	}); err != nil {
		return err
	}
	appResp := struct {
		Value []struct {
			Id string `json:"id,omitempty"`
		}
	}{}
	response := &rest.Response{
		Body: &appResp,
		Error: struct {
			Message *string `json:"message,omitempty"`
		}{},
	}	
	restApiClient := rest.NewRestApiClient(p.ctx, "https://graph.microsoft.com")
	err = restApiClient.NewRequest(
		&rest.Request{
			Path: "/v1.0/servicePrincipals",
			Headers: rest.NV{
				"Authorization": fmt.Sprintf("Bearer %s", token.Token),
			},
			RawQuery: fmt.Sprintf("$filter=%s",
				url.PathEscape(fmt.Sprintf("servicePrincipalNames/any(c:c eq '%s')", *config.ClientID)),
			),					
		},
	).DoGet(response)
	if err != nil {
		return err
	}

	if len(appResp.Value) == 0 {
		return fmt.Errorf(
			"unable to determine service principal for client id '%s'", 
			*config.ClientID,
		)
	}
	p.servicePrincipalID = appResp.Value[0].Id
	return nil
}

// Azure Provider helper function specific to the azure
// provider to determine the storage account name derived
// from the default resource group name.
func GetAzureStorageAccountName(p CloudProvider) string {

	config := p.(*azureProvider).cloudProvider.
		config.(*azureProviderConfig)

	storageAccountName := saNameRegex.ReplaceAllString(*config.DefaultResourceGroup, "")
	if len(storageAccountName) > 24 {
		// storage account names can only be 24 chars in length
		storageAccountName = storageAccountName[0:24]
	}
	return storageAccountName
}

// interface: config/Configurable functions of base cloud provider

func (p *azureProvider) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudProvider
	)

	if copy, err = newAzureProvider(); err != nil {
		return nil, err
	}

	config := p.cloudProvider.
		config.(*azureProviderConfig)
	configCopy := copy.(*azureProvider).cloudProvider.
		config.(*azureProviderConfig)

	configCopy.Environment = utils.CopyStrPtr(config.Environment)
	configCopy.SubscriptionID = utils.CopyStrPtr(config.SubscriptionID)
	configCopy.ClientID = utils.CopyStrPtr(config.ClientID)
	configCopy.ClientSecret = utils.CopyStrPtr(config.ClientSecret)
	configCopy.TenantID = utils.CopyStrPtr(config.TenantID)
	configCopy.DefaultResourceGroup = utils.CopyStrPtr(config.DefaultResourceGroup)
	configCopy.DefaultLocation = utils.CopyStrPtr(config.DefaultLocation)

	return copy, nil
}

func (p *azureProvider) IsValid() bool {

	config := p.cloudProvider.
		config.(*azureProviderConfig)

	return config.Environment != nil && len(*config.Environment) > 0 &&
		config.SubscriptionID != nil && len(*config.SubscriptionID) > 0 &&
		config.ClientID != nil && len(*config.ClientID) > 0 &&
		config.ClientSecret != nil && len(*config.ClientSecret) > 0 &&
		config.TenantID != nil && len(*config.TenantID) > 0
}

// interface: config/provider/CloudProvider functions

func (p *azureProvider) Connect() error {

	var (
		err error
		
		rscGrpClient *armresources.ResourceGroupsClient

		getresp armresources.ResourceGroupsClientGetResponse
		crtresp armresources.ResourceGroupsClientCreateOrUpdateResponse
	)

	if !p.IsValid() {
		return fmt.Errorf("provider configuration is not valid")
	}
	if !p.isInitialized {

		config := p.cloudProvider.
			config.(*azureProviderConfig)
		
		if err = p.connect(config); err != nil {
			return err
		}

		if rscGrpClient, err = armresources.NewResourceGroupsClient(
			*config.SubscriptionID, 
			p.clientCreds, 
			p.clientOpts,
		); err != nil {
			return err
		}

		if getresp, err = rscGrpClient.Get(p.ctx, *config.DefaultResourceGroup, nil); err != nil {

			// create default resource group
			if crtresp, err = rscGrpClient.CreateOrUpdate(p.ctx,
				*config.DefaultResourceGroup,
				armresources.ResourceGroup{
					Location: config.DefaultLocation,
				},
				nil,
			); err != nil {
				return err
			}			
			p.defaultResGrp = crtresp.ResourceGroup

		} else {
			p.defaultResGrp = getresp.ResourceGroup
		}

		p.isInitialized = true
	}
	return nil
}

func (p *azureProvider) Region() *string {

	config := p.cloudProvider.
		config.(*azureProviderConfig)

	return config.DefaultLocation
}

func (p *azureProvider) GetRegions() []RegionInfo {

	var (
		err error

		client *armsubscriptions.Client

		regionInfoList []RegionInfo
		resp           armsubscriptions.ClientListLocationsResponse
	)

	if p.isInitialized {
		if client, err = armsubscriptions.NewClient(p.clientCreds, nil); err == nil {
			
			config := p.cloudProvider.
				config.(*azureProviderConfig)

			listLocations := client.NewListLocationsPager(
				*config.SubscriptionID, 
				&armsubscriptions.ClientListLocationsOptions{
					IncludeExtendedLocations: to.Ptr(false),
				},
			)
			regionInfoList = []RegionInfo{}
			for listLocations.More() {
				if resp, err = listLocations.NextPage(p.ctx); err != nil {
					break
				}

				for _, l := range resp.LocationListResult.Value {
					if ( len(*l.Name) > 0 &&
						!strings.HasSuffix(*l.Name, "stage") && 
						!strings.HasSuffix(*l.Name, "stg") ) {

						regionInfoList = append(regionInfoList, RegionInfo{*l.Name, *l.DisplayName})
					}
				}
			}
			if err == nil {
				sortRegions(regionInfoList)
				logger.TraceMessage("Azure location list retrieved via API: %# v", regionInfoList)
	
				return regionInfoList	
			}
		}

		logger.DebugMessage("Unable to retrieve Azure locations via API call: %s", err.Error())
	}

	// The list is hard-coded as you need to be
	// authenticated to retrieve it via the API.
	// This list will be validated with the list
	// retrieved via the API call when the unit
	// tests are run and will fail if the region
	// list changes.

	regionInfoList = []RegionInfo{
		{"global", "Global"},
		{"unitedstates", "United States"},
		{"unitedstateseuap", "United States EUAP"},
		{"westus", "West US"},
		{"westus2", "West US 2"},
		{"westus3", "West US 3"},
		{"westcentralus", "West Central US"},
		{"northcentralus", "North Central US"},
		{"centralus", "Central US"},
		{"centraluseuap", "Central US EUAP"},
		{"eastus", "East US"},
		{"eastus2", "East US 2"},
		{"eastus2euap", "East US 2 EUAP"},
		{"canada", "Canada"},
		{"canadacentral", "Canada Central"},
		{"canadaeast", "Canada East"},
		{"brazil", "Brazil"},
		{"brazilsouth", "Brazil South"},
		{"brazilsoutheast", "Brazil Southeast"},
		{"europe", "Europe"},
		{"westeurope", "West Europe"},
		{"uk", "United Kingdom"},
		{"uksouth", "UK South"},
		{"ukwest", "UK West"},
		{"france", "France"},
		{"francecentral", "France Central"},
		{"francesouth", "France South"},
		{"germany", "Germany"},
		{"germanynorth", "Germany North"},
		{"germanywestcentral", "Germany West Central"},
		{"northeurope", "North Europe"},
		{"norway", "Norway"},
		{"norwayeast", "Norway East"},
		{"norwaywest", "Norway West"},
		{"qatarcentral", "Qatar Central"},
		{"singapore", "Singapore"},
		{"southafrica", "South Africa"},
		{"southafricanorth", "South Africa North"},
		{"southafricawest", "South Africa West"},
		{"southcentralus", "South Central US"},
		{"southeastasia", "Southeast Asia"},
		{"swedencentral", "Sweden Central"},
		{"switzerland", "Switzerland"},
		{"switzerlandnorth", "Switzerland North"},
		{"switzerlandwest", "Switzerland West"},
		{"uae", "United Arab Emirates"},
		{"uaecentral", "UAE Central"},
		{"uaenorth", "UAE North"},
		{"asia", "Asia"},
		{"india", "India"},
		{"westindia", "West India"},
		{"centralindia", "Central India"},
		{"southindia", "South India"},
		{"jioindiawest", "Jio India West"},
		{"jioindiacentral", "Jio India Central"},
		{"korea", "Korea"},
		{"koreacentral", "Korea Central"},
		{"koreasouth", "Korea South"},
		{"japan", "Japan"},
		{"japaneast", "Japan East"},
		{"japanwest", "Japan West"},
		{"asiapacific", "Asia Pacific"},
		{"eastasia", "East Asia"},
		{"australia", "Australia"},
		{"australiacentral", "Australia Central"},
		{"australiacentral2", "Australia Central 2"},
		{"australiaeast", "Australia East"},
		{"australiasoutheast", "Australia Southeast"},
	}
	sortRegions(regionInfoList)
	logger.TraceMessage("Pre-defined Azure location list: %# v", regionInfoList)

	return regionInfoList
}

func (p *azureProvider) GetCompute() (cloud.Compute, error) {

	if !p.isInitialized {
		return nil, fmt.Errorf("azure provider has not been initialized")
	}

	config := p.cloudProvider.
		config.(*azureProviderConfig)

	return cloud.NewAzureCompute(p.ctx,
		p.clientCreds,
		p.clientOpts,
		*config.DefaultResourceGroup,
		*config.DefaultLocation,
		*config.SubscriptionID,
	)
}

func (p *azureProvider) GetStorage() (cloud.Storage, error) {

	if !p.isInitialized {
		return nil, fmt.Errorf("azure provider has not been initialized")
	}

	config := p.cloudProvider.
		config.(*azureProviderConfig)

	return cloud.NewAzureStorage(p.ctx,
		p.clientCreds,
		p.clientOpts,
		GetAzureStorageAccountName(p),
		*config.DefaultResourceGroup,
		*config.DefaultLocation,
		*config.SubscriptionID,
		p.servicePrincipalID,
	)
}
