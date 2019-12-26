package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mevansam/goutils/utils"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/google/uuid"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	forms_config "github.com/mevansam/gocloud/forms"
)

type azureProvider struct {
	cloudProvider

	// background context
	ctx context.Context

	// indicates if provider client has
	// been prepared to make API requests
	isInitialized bool

	authorizer    *autorest.BearerAuthorizer
	defaultResGrp resources.Group
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

var environments = map[string]string{
	"public":       "AzurePublicCloud",
	"usgovernment": "AzureUSGovernmentCloud",
	"german":       "AzureGermanCloud",
	"china":        "AzureChinaCloud",
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
		Tags: []string{"provider", "target"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:         "default_location",
		DisplayName:  "Default Location or Region",
		Description:  "The location of the default resource group.",
		InputType:    forms.String,
		DefaultValue: utils.PtrToStr("eastus"),
		Tags:         []string{"provider", "target"},

		AcceptedValues:             regionList,
		AcceptedValuesErrorMessage: "Not a valid AWS region.",
	}); err != nil {
		return err
	}

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

		env         azure.Environment
		oauthConfig *adal.OAuthConfig
		token       *adal.ServicePrincipalToken
	)

	if !p.IsValid() {
		return fmt.Errorf("provider configuration is not valid")
	}
	config := p.cloudProvider.
		config.(*azureProviderConfig)

	if env, err = azure.EnvironmentFromName(
		environments[*config.Environment],
	); err != nil {
		return err
	}
	if oauthConfig, err = adal.NewOAuthConfig(
		env.ActiveDirectoryEndpoint,
		*config.TenantID,
	); err != nil {
		return err
	}
	if token, err = adal.NewServicePrincipalToken(
		*oauthConfig,
		*config.ClientID,
		*config.ClientSecret,
		env.ResourceManagerEndpoint,
	); err != nil {
		return err
	}

	p.authorizer = autorest.NewBearerAuthorizer(token)

	// ensure default resource group exists
	client := resources.NewGroupsClient(*config.SubscriptionID)
	client.Authorizer = p.authorizer
	client.AddToUserAgent(httpUserAgent)

	if p.defaultResGrp, err = client.Get(p.ctx,
		*config.DefaultResourceGroup,
	); err != nil {

		// create default resource group
		if p.defaultResGrp, err = client.CreateOrUpdate(p.ctx,
			*config.DefaultResourceGroup,
			resources.Group{
				Location: config.DefaultLocation,
			},
		); err != nil {
			return err
		}
	}

	p.isInitialized = true
	return nil
}

func (p *azureProvider) Region() *string {

	config := p.cloudProvider.
		config.(*azureProviderConfig)

	return config.DefaultLocation
}

func (p *azureProvider) GetRegions() []RegionInfo {

	var (
		err            error
		regionInfoList []RegionInfo
	)

	if p.isInitialized {

		var (
			result subscriptions.LocationListResult
		)

		client := subscriptions.NewClient()
		client.Authorizer = p.authorizer
		client.AddToUserAgent(httpUserAgent)

		config := p.cloudProvider.
			config.(*azureProviderConfig)

		result, err = client.ListLocations(p.ctx, *config.SubscriptionID)
		if err == nil {
			regionInfoList = []RegionInfo{}
			for _, l := range *result.Value {
				regionInfoList = append(regionInfoList, RegionInfo{*l.Name, *l.DisplayName})
			}
			sortRegions(regionInfoList)
			logger.TraceMessage("Azure location list retrieved via API: %# v", regionInfoList)

			return regionInfoList
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
		RegionInfo{"eastasia", "East Asia"},
		RegionInfo{"southeastasia", "Southeast Asia"},
		RegionInfo{"centralus", "Central US"},
		RegionInfo{"eastus", "East US"},
		RegionInfo{"eastus2", "East US 2"},
		RegionInfo{"westus", "West US"},
		RegionInfo{"northcentralus", "North Central US"},
		RegionInfo{"southcentralus", "South Central US"},
		RegionInfo{"northeurope", "North Europe"},
		RegionInfo{"westeurope", "West Europe"},
		RegionInfo{"japanwest", "Japan West"},
		RegionInfo{"japaneast", "Japan East"},
		RegionInfo{"brazilsouth", "Brazil South"},
		RegionInfo{"australiaeast", "Australia East"},
		RegionInfo{"australiasoutheast", "Australia Southeast"},
		RegionInfo{"southindia", "South India"},
		RegionInfo{"centralindia", "Central India"},
		RegionInfo{"westindia", "West India"},
		RegionInfo{"canadacentral", "Canada Central"},
		RegionInfo{"canadaeast", "Canada East"},
		RegionInfo{"uksouth", "UK South"},
		RegionInfo{"ukwest", "UK West"},
		RegionInfo{"westcentralus", "West Central US"},
		RegionInfo{"westus2", "West US 2"},
		RegionInfo{"koreacentral", "Korea Central"},
		RegionInfo{"koreasouth", "Korea South"},
		RegionInfo{"francecentral", "France Central"},
		RegionInfo{"francesouth", "France South"},
		RegionInfo{"australiacentral", "Australia Central"},
		RegionInfo{"australiacentral2", "Australia Central 2"},
		RegionInfo{"uaecentral", "UAE Central"},
		RegionInfo{"uaenorth", "UAE North"},
		RegionInfo{"southafricanorth", "South Africa North"},
		RegionInfo{"southafricawest", "South Africa West"},
		RegionInfo{"switzerlandnorth", "Switzerland North"},
		RegionInfo{"switzerlandwest", "Switzerland West"},
		RegionInfo{"germanynorth", "Germany North"},
		RegionInfo{"germanywestcentral", "Germany West Central"},
		RegionInfo{"norwaywest", "Norway West"},
		RegionInfo{"norwayeast", "Norway East"},
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
		p.authorizer,
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
		p.authorizer,
		GetAzureStorageAccountName(p),
		*config.DefaultResourceGroup,
		*config.DefaultLocation,
		*config.SubscriptionID,
	)
}
