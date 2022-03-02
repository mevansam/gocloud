package cloud

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mevansam/goutils/logger"

	"github.com/aws/aws-sdk-go/aws/session"
)

type AWSComputeProperties struct {
	FilterTags map[string]string
}

type awsCompute struct {
	session *session.Session

	props AWSComputeProperties
}

type awsComputeInstance struct {
	session  *session.Session
	instance *ec2.Instance
	name     string
}

func NewAWSCompute(
	session *session.Session,
) (Compute, error) {

	return &awsCompute{
		session: session,

		props: AWSComputeProperties{
			FilterTags: make(map[string]string),
		},
	}, nil
}

func (c *awsCompute) newAWSComputeInstance(
	session *session.Session,
	instance *ec2.Instance,
) (ComputeInstance, error) {

	for _, v := range instance.Tags {
		if *v.Key == "Name" {
			return &awsComputeInstance{
				session:  session,
				instance: instance,
				name:     *v.Value,
			}, nil
		}
	}

	return nil, fmt.Errorf(
		fmt.Sprintf("name not found for instance with id '%s'.", *instance.InstanceId),
	)
}

// interface: cloud/Compute implementation

func (c *awsCompute) SetProperties(props interface{}) {

	p := props.(AWSComputeProperties)
	if p.FilterTags != nil {
		c.props.FilterTags = p.FilterTags
	}
}

func (c *awsCompute) GetInstance(name string) (ComputeInstance, error) {

	var (
		err error

		describeResult *ec2.DescribeInstancesOutput
	)
	svc := ec2.New(c.session)

	filters := make([]*ec2.Filter, 2, len(c.props.FilterTags)+2)
	filters[0] = &ec2.Filter{
		Name:   aws.String("tag:Name"),
		Values: []*string{aws.String(name)},
	}
	filters[1] = &ec2.Filter{
		Name: aws.String("instance-state-name"),
		Values: []*string{
			aws.String("pending"),
			aws.String("running"),
			aws.String("shutting-down"),
			aws.String("stopping"),
			aws.String("stopped"),
		},
	}
	for t, v := range c.props.FilterTags {
		filters = append(filters,
			&ec2.Filter{
				Name:   aws.String("tag:" + t),
				Values: []*string{aws.String(v)},
			},
		)
	}

	if describeResult, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filters,
	}); err != nil {
		return nil, err
	}
	if describeResult.Reservations != nil && len(describeResult.Reservations) == 1 {

		instances := (*describeResult.Reservations[0]).Instances
		if len(instances) == 1 {

			return c.newAWSComputeInstance(
				c.session,
				instances[0],
			)
		}
	}

	logger.DebugMessage(
		"Found none or more than one instance reservations with tag Name='%s': %# v",
		name, describeResult.Reservations,
	)
	return nil, fmt.Errorf(
		fmt.Sprintf("exactly one instance within a reservation with tag Name='%s' was not found", name),
	)
}

func (c *awsCompute) GetInstances(ids []string) ([]ComputeInstance, error) {

	numIds := len(ids)

	idFilter := &ec2.Filter{
		Name:   aws.String("instance-id"),
		Values: make([]*string, numIds),
	}
	for i, id := range ids {
		idFilter.Values[i] = aws.String(id)
	}

	filters := make([]*ec2.Filter, 1, 2)
	filters[0] = idFilter

	return c.listInstances(filters)
}

func (c *awsCompute) ListInstances() ([]ComputeInstance, error) {

	filters := make([]*ec2.Filter, 0, len(c.props.FilterTags)+1)
	for t, v := range c.props.FilterTags {
		filters = append(filters,
			&ec2.Filter{
				Name:   aws.String("tag:" + t),
				Values: []*string{aws.String(v)},
			},
		)
	}
	return c.listInstances(filters)
}

func (c *awsCompute) listInstances(filters []*ec2.Filter) ([]ComputeInstance, error) {

	var (
		err error

		describeResult *ec2.DescribeInstancesOutput
		instance       ComputeInstance
	)
	svc := ec2.New(c.session)
	computeInstances := []ComputeInstance{}

	// ensure terminated instances are
	// not included in the response
	filters = append(filters, &ec2.Filter{
		Name: aws.String("instance-state-name"),
		Values: []*string{
			aws.String("pending"),
			aws.String("running"),
			aws.String("shutting-down"),
			aws.String("stopping"),
			aws.String("stopped"),
		},
	})
	for t, v := range c.props.FilterTags {
		filters = append(filters,
			&ec2.Filter{
				Name:   aws.String("tag:" + t),
				Values: []*string{aws.String(v)},
			},
		)
	}

	if describeResult, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filters,
	}); err != nil {
		return nil, err
	}
	if describeResult.Reservations != nil {
		for _, r := range describeResult.Reservations {
			if r.Instances != nil {
				for _, i := range r.Instances {

					if instance, err = c.newAWSComputeInstance(
						c.session, i,
					); err != nil {
						return nil, err
					}
					computeInstances = append(computeInstances, instance)
				}
			}
		}
	}

	return computeInstances, nil
}

// interface: cloud/ComputeInstance implementation

func (c *awsComputeInstance) ID() string {
	return *c.instance.InstanceId
}

func (c *awsComputeInstance) Name() string {
	return c.name
}

func (c *awsComputeInstance) PublicIP() string {
	if c.instance.PublicIpAddress != nil {
		return *c.instance.PublicIpAddress
	} else {
		return ""
	}
}

func (c *awsComputeInstance) State() (InstanceState, error) {

	var (
		err error

		describeResult *ec2.DescribeInstancesOutput
	)
	svc := ec2.New(c.session)

	if describeResult, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{c.instance.InstanceId},
	}); err != nil {
		return StateUnknown, err
	}
	if describeResult.Reservations == nil || len(describeResult.Reservations) == 0 ||
		(*describeResult.Reservations[0]).Instances == nil || len((*describeResult.Reservations[0]).Instances) == 0 {

		return StateUnknown, fmt.Errorf(
			fmt.Sprintf(
				"unable to retrieve state for instance with id '%s', as it was not found",
				*c.instance.InstanceId),
		)
	}
	c.instance = (*describeResult.Reservations[0]).Instances[0]

	switch *c.instance.State.Name {
	case "running":
		return StateRunning, nil
	case "stopped":
		return StateStopped, nil
	case "pending", "shutting-down", "stopping":
		return StatePending, nil
	}

	return StateUnknown, nil
}

func (c *awsComputeInstance) Start() error {

	var (
		err error
	)
	svc := ec2.New(c.session)

	if _, err = svc.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: []*string{c.instance.InstanceId},
	}); err != nil {
		return err
	}
	if err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{c.instance.InstanceId},
	}); err != nil {
		return err
	}
	return nil
}

func (c *awsComputeInstance) Restart() error {

	var (
		err error
	)
	svc := ec2.New(c.session)

	if _, err = svc.RebootInstances(&ec2.RebootInstancesInput{
		InstanceIds: []*string{c.instance.InstanceId},
	}); err != nil {
		return err
	}
	if err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{c.instance.InstanceId},
	}); err != nil {
		return err
	}
	return nil
}

func (c *awsComputeInstance) Stop() error {

	var (
		err error
	)
	svc := ec2.New(c.session)

	if _, err = svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{c.instance.InstanceId},
	}); err != nil {
		return err
	}
	if err = svc.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{c.instance.InstanceId},
	}); err != nil {
		return err
	}
	return nil
}

func (c *awsComputeInstance) CanConnect(port int) bool {

	publicIP := c.PublicIP()
	if publicIP != "" {
		return canConnect(
			fmt.Sprintf("%s:%d", publicIP, port),
		)
	} else {
		return false
	}
}
