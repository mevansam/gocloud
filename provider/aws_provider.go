package provider

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/utils"

	forms_config "github.com/mevansam/gocloud/forms"
)

type awsProvider struct {
	cloudProvider

	// indicates if provider client has
	// been prepared to make API requests
	isInitialized bool

	session *session.Session
}

type awsProviderConfig struct {
	AccessKey *string `json:"access_key,omitempty" form_field:"access_key"`
	SecretKey *string `json:"secret_key,omitempty" form_field:"secret_key"`
	Region    *string `json:"region,omitempty" form_field:"region"`
	Token     *string `json:"token,omitempty" form_field:"token"`
}

func newAWSProvider() (CloudProvider, error) {

	var (
		err            error
		providerConfig awsProviderConfig
	)

	provider := &awsProvider{
		cloudProvider: cloudProvider{
			name:   "aws",
			config: &providerConfig,
		},
		isInitialized: false,
	}
	err = provider.createAWSInputForm()
	return provider, err
}

func (p *awsProvider) createAWSInputForm() error {

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

	form = forms_config.CloudConfigForms.NewGroup(p.name, "Amazon Web Services Cloud Platform")

	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "access_key",
		DisplayName: "Access Key",
		Description: "The AWS user account's access key id.",
		InputType:   forms.String,
		EnvVars: []string{
			"AWS_ACCESS_KEY_ID",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "secret_key",
		DisplayName: "Secret Key",
		Description: "The AWS user account's secret key.",
		InputType:   forms.String,
		Sensitive:   true,
		EnvVars: []string{
			"AWS_SECRET_ACCESS_KEY",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:        "token",
		DisplayName: "Token",
		Description: "AWS multi-factor authentication token.",
		InputType:   forms.String,
		Sensitive:   true,
		EnvVars: []string{
			"AWS_SESSION_TOKEN",
		},
		Tags: []string{"provider"},
	}); err != nil {
		return err
	}
	if _, err = form.NewInputField(forms.FieldAttributes{
		Name:         "region",
		DisplayName:  "Region",
		Description:  "The AWS region to create resources in.",
		InputType:    forms.String,
		DefaultValue: utils.PtrToStr("us-east-1"),
		EnvVars: []string{
			"AWS_DEFAULT_REGION",
		},
		Tags:                       []string{"provider", "target"},
		AcceptedValues:             regionList,
		AcceptedValuesErrorMessage: "Not a valid AWS region.",
	}); err != nil {
		return err
	}

	return nil
}

// interface: config/Configurable functions of base cloud provider

func (p *awsProvider) Copy() (config.Configurable, error) {

	var (
		err error

		copy CloudProvider
	)

	if copy, err = newAWSProvider(); err != nil {
		return nil, err
	}

	config := p.cloudProvider.
		config.(*awsProviderConfig)
	configCopy := copy.(*awsProvider).cloudProvider.
		config.(*awsProviderConfig)

	configCopy.AccessKey = utils.CopyStrPtr(config.AccessKey)
	configCopy.SecretKey = utils.CopyStrPtr(config.SecretKey)
	configCopy.Region = utils.CopyStrPtr(config.Region)
	configCopy.Token = utils.CopyStrPtr(config.Token)

	return copy, nil
}

func (p *awsProvider) IsValid() bool {

	config := p.cloudProvider.
		config.(*awsProviderConfig)

	return config.Region != nil && len(*config.Region) > 0 &&
		config.AccessKey != nil && len(*config.AccessKey) > 0 &&
		config.SecretKey != nil && len(*config.SecretKey) > 0
}

// interface: config/provider/CloudProvider functions

func (p *awsProvider) Connect() error {

	var (
		err error
	)

	if !p.IsValid() {
		return fmt.Errorf("provider configuration is not valid")
	}
	config := p.cloudProvider.
		config.(*awsProviderConfig)

	token := ""
	if config.Token != nil {
		token = *config.Token
	}

	p.session, err = session.NewSession(&aws.Config{
		Region: aws.String(*config.Region),
		Credentials: credentials.NewStaticCredentials(
			*config.AccessKey,
			*config.SecretKey,
			token,
		),
	})
	p.isInitialized = true
	return err
}

func (p *awsProvider) Regions() []RegionInfo {

	regionInfoList := []RegionInfo{}
	for _, r := range endpoints.AwsPartition().Regions() {
		regionInfoList = append(regionInfoList,
			RegionInfo{
				Name:        r.ID(),
				Description: r.Description(),
			})
	}
	sortRegions(regionInfoList)
	return regionInfoList
}

func (p *awsProvider) GetCompute() (cloud.Compute, error) {

	if !p.isInitialized {
		return nil, fmt.Errorf("aws provider has not been initialized")
	}

	return cloud.NewAWSCompute(
		p.session,
	)
}

func (p *awsProvider) GetStorage() (cloud.Storage, error) {

	if !p.isInitialized {
		return nil, fmt.Errorf("aws provider has not been initialized")
	}

	config := p.cloudProvider.
		config.(*awsProviderConfig)

	return cloud.NewAWSStorage(
		p.session,
		*config.Region,
	)
}
