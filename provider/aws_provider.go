package provider

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goutils/config"
	"github.com/mevansam/goutils/forms"
)

type awsProvider struct {
	cloudProvider

	// indicates if provider client has
	// been prepared to make API requests
	isInitialized bool

	session *session.Session
}

type awsProviderConfig struct {
	AccessKey *string `json:"access_key,omitempty"`
	SecretKey *string `json:"secret_key,omitempty"`
	Region    *string `json:"region,omitempty"`
	Token     *string `json:"token,omitempty"`
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
	clougConfig := CloudConfigForms
	if clougConfig.HasGroup(p.name) {
		return nil
	}

	var (
		err   error
		form  *forms.InputGroup
		field forms.Input
	)

	regions := p.Regions()
	rr := make([]string, len(regions))
	for i, r := range regions {
		rr[i] = r.Name
	}

	form = CloudConfigForms.NewGroup(p.name, "Amazon Web Services Cloud Platform")

	if _, err = form.NewInputGroupField(
		/* name */ "access_key",
		/* displayName */ "Access Key",
		/* description */ "The AWS user account's access key id.",
		/* groupId */ 0,
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{
			"AWS_ACCESS_KEY_ID",
		},
		/* dependsOn */ []string{},
	); err != nil {
		return err
	}
	if field, err = form.NewInputField(
		/* name */ "secret_key",
		/* displayName */ "Secret Key",
		/* description */ "The AWS user account's secret key.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{
			"AWS_SECRET_ACCESS_KEY",
		},
		/* dependsOn */ []string{},
	); err != nil {
		return err
	}
	field.(*forms.InputField).SetSensitive(true)

	if field, err = form.NewInputField(
		/* name */ "region",
		/* displayName */ "Region",
		/* description */ "The AWS region to create resources in.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{
			"AWS_DEFAULT_REGION",
		},
		/* dependsOn */ []string{},
	); err != nil {
		return err
	}
	field.(*forms.InputField).SetAcceptedValues(&rr, "Not a valid AWS region.")

	if field, err = form.NewInputField(
		/* name */ "token",
		/* displayName */ "Token",
		/* description */ "AWS multi-factor authentication token.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{
			"AWS_SESSION_TOKEN",
		},
		/* dependsOn */ []string{},
	); err != nil {
		return err
	}
	field.(*forms.InputField).SetSensitive(true)

	return nil
}

// interface: config/Configurable functions for base cloud provider

func (p *awsProvider) InputForm() (forms.InputForm, error) {

	var (
		err error

		field          *forms.InputField
		providerConfig *awsProviderConfig
	)

	// Bind AWS configuration data instance to input form
	form := CloudConfigForms.Group(p.name)
	providerConfig = p.cloudProvider.
		config.(*awsProviderConfig)

	field, _ = form.GetInputField("access_key")
	if err = field.SetValueRef(&providerConfig.AccessKey); err != nil {
		return nil, err
	}
	field, _ = form.GetInputField("secret_key")
	if err = field.SetValueRef(&providerConfig.SecretKey); err != nil {
		return nil, err
	}
	field, _ = form.GetInputField("region")
	if err = field.SetValueRef(&providerConfig.Region); err != nil {
		return nil, err
	}
	field, _ = form.GetInputField("token")
	if err = field.SetValueRef(&providerConfig.Token); err != nil {
		return nil, err
	}

	return form, nil
}

func (p *awsProvider) GetValue(name string) (*string, error) {

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

	*configCopy = *config

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
