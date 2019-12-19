package provider

import (
	"context"
	"fmt"

	"github.com/mevansam/goutils/utils"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	compute "google.golang.org/api/compute/v1"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	forms_config "github.com/mevansam/gocloud/forms"
)

type googleProvider struct {
	cloudProvider

	// background context
	ctx context.Context

	// indicates if provider client has
	// been prepared to make API requests
	isInitialized bool

	computeService *compute.Service
	storageClient  *storage.Client
}

type googleProviderConfig struct {
	Authentication struct {
		Credentials *string `json:"credentials,omitempty" form_field:"credentials"`
		AccessToken *string `json:"access_token,omitempty" form_field:"access_token"`
	} `json:"authentication"`

	Project *string `json:"project,omitempty" form_field:"project"`
	Region  *string `json:"region,omitempty" form_field:"region"`
	Zone    *string `json:"zone,omitempty" form_field:"zone"`
}

var staticGoogleRegionMap = map[string]string{
	"asia-east1":              "Changhua County, Taiwan",
	"asia-east2":              "Hong Kong",
	"asia-northeast1":         "Tokyo, Japan",
	"asia-northeast2":         "Osaka, Japan",
	"asia-south1":             "Mumbai, India",
	"asia-southeast1":         "Jurong West, Singapore",
	"australia-southeast1":    "Sydney, Australia",
	"europe-north1":           "Hamina, Finland",
	"europe-west1":            "St. Ghislain, Belgium",
	"europe-west2":            "London, England, UK",
	"europe-west3":            "Frankfurt, Germany",
	"europe-west4":            "Eemshaven, Netherlands",
	"europe-west6":            "Zürich, Switzerland",
	"northamerica-northeast1": "Montréal, Québec, Canada",
	"southamerica-east1":      "São Paulo, Brazil",
	"us-central1":             "Council Bluffs, Iowa, USA",
	"us-east1":                "Moncks Corner, South Carolina, USA",
	"us-east4":                "Ashburn, Northern Virginia, USA",
	"us-west1":                "The Dalles, Oregon, USA",
	"us-west2":                "Los Angeles, California, USA",
}

func newGoogleProvider() (CloudProvider, error) {

	var (
		err            error
		providerConfig googleProviderConfig
	)

	provider := &googleProvider{
		cloudProvider: cloudProvider{
			name:   "google",
			config: &providerConfig,
		},
		ctx:           context.Background(),
		isInitialized: false,
	}
	err = provider.createGoogleInputForm()
	return provider, err
}

func (p *googleProvider) createGoogleInputForm() error {

	// Do not recreate form template if it exists
	clougConfig := forms_config.CloudConfigForms
	if clougConfig.HasGroup(p.name) {
		return nil
	}

	var (
		err  error
		form *forms.InputGroup
	)

	regions := p.Regions()
	regionList := make([]string, len(regions))
	for i, r := range regions {
		regionList[i] = r.Name
	}

	form = forms_config.CloudConfigForms.NewGroup(p.name, "Google Cloud Platform")

	form.NewInputContainer(
		/* name */ "authentication",
		/* displayName */ "Authentication",
		/* description */ "Google Cloud authentication credentials",
		/* groupId */ 1,
	)

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:          "credentials",
		DisplayName:   "Credentials",
		Description:   "The contents of a service account key file in JSON format.",
		GroupID:       1,
		InputType:     forms.String,
		ValueFromFile: true,
		EnvVars: []string{
			"GOOGLE_CREDENTIALS",
			"GOOGLE_CLOUD_KEYFILE_JSON",
			"GCLOUD_KEYFILE_JSON",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "access_token",
		DisplayName: "Access Token",
		Description: "A temporary OAuth 2.0 access token obtained from the Google Authorization server.",
		GroupID:     1,
		InputType:   forms.String,
		Sensitive:   true,
		EnvVars: []string{
			"GOOGLE_OAUTH_ACCESS_TOKEN",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "project",
		DisplayName: "Project",
		Description: "The Google Cloud Platform project to manage resources in.",
		InputType:   forms.String,
		EnvVars: []string{
			"GOOGLE_PROJECT",
			"GOOGLE_CLOUD_PROJECT",
			"GCLOUD_PROJECT",
			"CLOUDSDK_CORE_PROJECT",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if err = form.AddFieldValueHint("project", "field://credentials/project_id"); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "region",
		DisplayName: "Region",
		Description: "The default region to manage resources in.",
		InputType:   forms.String,
		EnvVars: []string{
			"GOOGLE_REGION",
			"GCLOUD_REGION",
			"CLOUDSDK_COMPUTE_REGION",
		},
		Tags: []string{"provider", "target"},

		AcceptedValues:             regionList,
		AcceptedValuesErrorMessage: "Not a valid Google Cloud region.",
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "zone",
		DisplayName: "Zone",
		Description: "The default zone to manage resources in.",
		InputType:   forms.String,
		EnvVars: []string{
			"GOOGLE_ZONE",
			"GCLOUD_ZONE",
			"CLOUDSDK_COMPUTE_ZONE",
		},
		Tags: []string{"provider", "target"},
	}); err != nil {
		return err
	}

	return nil
}

// interface: config/Configurable functions of base cloud provider

func (p *googleProvider) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudProvider
	)

	if copy, err = newGoogleProvider(); err != nil {
		return nil, err
	}

	config := p.cloudProvider.
		config.(*googleProviderConfig)
	configCopy := copy.(*googleProvider).cloudProvider.
		config.(*googleProviderConfig)

	configCopy.Authentication.Credentials = utils.CopyStrPtr(config.Authentication.Credentials)
	configCopy.Authentication.AccessToken = utils.CopyStrPtr(config.Authentication.AccessToken)
	configCopy.Project = utils.CopyStrPtr(config.Project)
	configCopy.Region = utils.CopyStrPtr(config.Region)
	configCopy.Zone = utils.CopyStrPtr(config.Zone)

	return copy, nil
}

func (p *googleProvider) IsValid() bool {

	config := p.cloudProvider.
		config.(*googleProviderConfig)

	return config.Project != nil && len(*config.Project) > 0 &&
		config.Region != nil && len(*config.Region) > 0 &&
		((config.Authentication.Credentials != nil && len(*config.Authentication.Credentials) > 0) ||
			(config.Authentication.AccessToken != nil && len(*config.Authentication.AccessToken) > 0))
}

// interface: config/provider/CloudProvider functions

func (p *googleProvider) Connect() error {

	var (
		err error
	)

	if !p.IsValid() {
		return fmt.Errorf("provider configuration is not valid")
	}
	config := p.cloudProvider.
		config.(*googleProviderConfig)

	if p.computeService, err = compute.NewService(
		p.ctx,
		option.WithCredentialsJSON([]byte(*config.Authentication.Credentials)),
	); err != nil {
		return err
	}
	if p.storageClient, err = storage.NewClient(
		p.ctx,
		option.WithCredentialsJSON([]byte(*config.Authentication.Credentials)),
	); err != nil {
		return err
	}

	p.isInitialized = true
	return nil
}

func (p *googleProvider) Regions() []RegionInfo {

	var (
		err error
	)

	regionInfoList := []RegionInfo{}

	if p.isInitialized {

		config := p.cloudProvider.
			config.(*googleProviderConfig)

		if err = p.computeService.Regions.List(*config.Project).
			Pages(p.ctx,
				func(page *compute.RegionList) error {
					for _, region := range page.Items {
						regionInfoList = append(regionInfoList,
							RegionInfo{
								Name:        region.Name,
								Description: region.Description,
							},
						)
					}
					return nil
				},
			); err == nil {
			sortRegions(regionInfoList)
			logger.TraceMessage("Google region list retrieved via API: %# v", regionInfoList)

			return regionInfoList
		}

		logger.DebugMessage("Unable to retrieve Google region via API call: %s", err.Error())
	}

	// The list is hard-coded as you need to be
	// authenticated to retrieve it via the API.
	// This list will be validated with the list
	// retrieved via the API call when the unit
	// tests are run and will fail if the region
	// list changes.

	for region, description := range staticGoogleRegionMap {
		regionInfoList = append(regionInfoList,
			RegionInfo{
				Name:        region,
				Description: description,
			},
		)
	}

	sortRegions(regionInfoList)
	return regionInfoList
}

func (p *googleProvider) GetCompute() (cloud.Compute, error) {

	config := p.cloudProvider.
		config.(*googleProviderConfig)

	return cloud.NewGoogleCompute(
		p.computeService,
		*config.Project,
		*config.Region,
		*config.Zone,
	)
}

func (p *googleProvider) GetStorage() (cloud.Storage, error) {

	config := p.cloudProvider.
		config.(*googleProviderConfig)

	return cloud.NewGoogleStorage(p.ctx, p.storageClient, *config.Project, *config.Region)
}
