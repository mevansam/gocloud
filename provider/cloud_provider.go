package provider

import (
	"encoding/json"
	"sort"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"

	forms_config "github.com/mevansam/gocloud/forms"
)

// interface for a configurable cloud provider
type CloudProvider interface {
	config.Configurable

	// Connects to the cloud provider using the
	// configured credentials
	Connect() error

	// The currently configured region for this provider
	Region() *string

	// List of regions for the client. If the provider
	// instances is not configured then a static list
	// is returned. Otherwise the list retrieved from
	// the cloud API is returned.
	GetRegions() []RegionInfo

	// Returns the provider's compute entity
	GetCompute() (cloud.Compute, error)

	// Returns the provider's storage entity
	GetStorage() (cloud.Storage, error)

	// adds the provider configuration variables
	// to the given variable map
	GetVars(vars map[string]string) error	
}

// base cloud provider implementation
type cloudProvider struct {
	name   string
	config interface{}
}

type newProvider func() (CloudProvider, error)

var providerNames = map[string]newProvider{
	"aws":    newAWSProvider,
	"azure":  newAzureProvider,
	"google": newGoogleProvider,
}

// in: the iaas to create a cloud provider configuration template for
// out: a cloud provider configuration template
func NewCloudProvider(iaas string) (CloudProvider, error) {

	var (
		newProvider newProvider
		exists      bool
	)

	if newProvider, exists = providerNames[iaas]; !exists {
		return newNullProvider(iaas)
	}
	return newProvider()
}

// out: a map of included cloud provider templates
func NewCloudProviderTemplates() (map[string]CloudProvider, error) {

	var (
		err error
	)

	templates := make(map[string]CloudProvider)
	for iaas, newProvider := range providerNames {
		if templates[iaas], err = newProvider(); err != nil {
			return nil, err
		}
	}
	return templates, nil
}

// sorts the given slice of cloud provider
// structs in ascending order of name
func SortCloudProviders(providers []CloudProvider) {
	sort.Sort(&cloudProviderSorter{providers})
}

// interface: config/Configurable functions for base cloud provider

func (p *cloudProvider) Name() string {
	return p.name
}

func (p *cloudProvider) Description() string {
	return forms_config.CloudConfigForms.Group(p.name).Description()
}

func (p *cloudProvider) InputForm() (forms.InputForm, error) {

	var (
		err error
	)

	form := forms_config.CloudConfigForms.Group(p.name)
	if err = form.BindFields(p.config); err != nil {
		return nil, err
	}
	return form, nil
}

func (p *cloudProvider) GetValue(name string) (*string, error) {

	var (
		err error

		form  forms.InputForm
		field *forms.InputField
	)

	if form, err = p.InputForm(); err != nil {
		return nil, err
	}
	if field, err = form.GetInputField(name); err != nil {
		return nil, err
	}
	return field.Value(), nil
}

func (p *cloudProvider) Reset() {
}

// interface: provider/CloudProvider functions

func (p *cloudProvider) GetVars(vars map[string]string) error {

	var (
		err error

		form  forms.InputForm
		field *forms.InputField

		value *string
	)

	if form, err = p.InputForm(); err != nil {
		return err
	}
	if form != nil {
		for _, field = range form.InputFields() {
			if value = field.Value(); value != nil {
				for _, envVar := range field.EnvVars() {
					vars[envVar] = *value
				}
			}
		}		
	}
	return nil
}

// interface: encoding/json/Unmarshaler

func (p *cloudProvider) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &p.config)
}

// interface: encoding/json/Marshaler

func (p *cloudProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(&p.config)
}

// interface: sort/Interface

// sorter struct containa the slice of providers
// to be sorted and implements the sort.Interface
type cloudProviderSorter struct {
	providers []CloudProvider
}

// Len is part of sort.Interface.
func (cps *cloudProviderSorter) Len() int {
	return len(cps.providers)
}

// Swap is part of sort.Interface.
func (cps *cloudProviderSorter) Swap(i, j int) {
	cps.providers[i], cps.providers[j] = cps.providers[j], cps.providers[i]
}

// Less is part of sort.Interface. It is implemented
// by calling the "by" closure in the sorter.
func (cps *cloudProviderSorter) Less(i, j int) bool {
	return cps.providers[i].Name() < cps.providers[j].Name()
}
