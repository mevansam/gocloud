package cloud

import (
	"fmt"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
	compute "google.golang.org/api/compute/v1"
)

type GoogleComputeProperties struct {
	Region,
	Zone string

	// timeout for start/stop operations
	OpTimeout time.Duration

	FilterLabels map[string]string
}

type googleCompute struct {
	service   *compute.Service
	projectID string

	props GoogleComputeProperties
}

type googleComputeInstance struct {
	service  *compute.Service
	instance *compute.Instance

	projectID string
	zone      string

	props *GoogleComputeProperties
}

func NewGoogleCompute(
	service *compute.Service,
	projectID string,
	region string,
) (Compute, error) {

	return &googleCompute{
		service:   service,
		projectID: projectID,

		props: GoogleComputeProperties{
			Region: region,

			// 5 minute timeout for
			// start/stop operatons
			OpTimeout: time.Minute * 5,

			FilterLabels: make(map[string]string),
		},
	}, nil
}

func (c *googleCompute) zoneList() ([]string, error) {

	var (
		err error

		zoneList *compute.ZoneList
		zones    []string
	)

	if len(c.props.Zone) == 0 {
		// search all zones for a vm
		// matching the given name
		if zoneList, err = c.service.Zones.List(c.projectID).Do(); err != nil {
			return nil, err
		}
		zones = make([]string, len(zoneList.Items))
		for i, z := range zoneList.Items {
			zones[i] = z.Name
		}

	} else {
		zones = []string{c.props.Zone}
	}
	return zones, nil
}

func (c *googleComputeInstance) waitForState(state, etag string) error {

	var (
		err error
		wg  sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		instance := c.instance
		for {
			if instance, err = c.service.Instances.Get(
				c.projectID,
				c.zone,
				c.instance.Name,
			).IfNoneMatch(etag).Do(); err != nil {
				return
			}
			if instance.Status == state {
				c.instance = instance
				break
			}
			// pause for 1s
			time.Sleep(time.Second)
		}
	}()

	if utils.WaitTimeout(&wg, c.props.OpTimeout) {
		return err
	} else {
		return fmt.Errorf(
			fmt.Sprintf(
				"instance '%s' timed out waiting for state '%s'",
				c.instance.Name, state,
			),
		)
	}
}

// interface: cloud/Compute implementation

func (c *googleCompute) SetProperties(props interface{}) {

	p := props.(GoogleComputeProperties)
	if len(p.Region) > 0 {
		c.props.Region = p.Region
		c.props.Zone = p.Zone
	}
	if p.OpTimeout != 0 {
		c.props.OpTimeout = p.OpTimeout
	}
	if p.FilterLabels != nil {
		c.props.FilterLabels = p.FilterLabels
	}
}

func (c *googleCompute) GetInstance(name string) (ComputeInstance, error) {

	var (
		err error

		zones    []string
		instance *compute.Instance
	)

	if zones, err = c.zoneList(); err != nil {
		return nil, err
	}
	for _, z := range zones {
		if instance, err = c.service.Instances.Get(c.projectID, z, name).Do(); err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return &googleComputeInstance{
		service:  c.service,
		instance: instance,

		projectID: c.projectID,
		zone:      path.Base(instance.Zone),

		props: &c.props,
	}, nil
}

func (c *googleCompute) ListInstances() ([]ComputeInstance, error) {

	var (
		err error

		zones []string

		instanceList *compute.InstanceList
		call         *compute.InstancesListCall
	)

	instances := []ComputeInstance{}

	if zones, err = c.zoneList(); err != nil {
		return nil, err
	}
	for _, z := range zones {

		call = c.service.Instances.List(c.projectID, z)
		for label, value := range c.props.FilterLabels {
			call.Filter(
				fmt.Sprintf("labels.%s=%s", label, value),
			)
		}
		if instanceList, err = call.Do(); err != nil {
			return nil, err
		}
		if instanceList.Items != nil {
			for _, instance := range instanceList.Items {

				instances = append(instances,
					&googleComputeInstance{
						service:  c.service,
						instance: instance,

						projectID: c.projectID,
						zone:      path.Base(instance.Zone),

						props: &c.props,
					},
				)
			}
		}
	}
	return instances, nil
}

// interface: cloud/ComputeInstance implementation

func (c *googleComputeInstance) ID() string {
	return strconv.FormatUint(c.instance.Id, 10)
}

func (c *googleComputeInstance) Name() string {
	return c.instance.Name
}

func (c *googleComputeInstance) PublicIP() string {

	if len(c.instance.NetworkInterfaces) > 0 &&
		len(c.instance.NetworkInterfaces[0].AccessConfigs) > 0 {

		return c.instance.NetworkInterfaces[0].AccessConfigs[0].NatIP
	} else {
		return ""
	}
}

func (c *googleComputeInstance) State() (InstanceState, error) {

	var (
		err error

		instance *compute.Instance
	)

	// refresh instance detail
	if instance, err = c.service.Instances.Get(
		c.projectID,
		c.zone,
		c.instance.Name,
	).Do(); err != nil {
		return StateUnknown, err
	}

	c.instance = instance
	switch instance.Status {
	case "RUNNING":
		return StateRunning, nil
	case "TERMINATED":
		return StateStopped, nil
	case "PROVISIONING",
		"STAGING",
		"STOPPING",
		"REPAIRING":
		return StatePending, nil
	default:
		return StateUnknown, nil
	}
}

func (c *googleComputeInstance) Start() error {

	var (
		err error

		operation *compute.Operation
	)

	logger.TraceMessage(
		"Starting instance '%s'.", c.instance.Name)

	if operation, err = c.service.Instances.Start(
		c.projectID,
		c.zone,
		c.instance.Name,
	).Do(); err != nil {
		return err
	}

	return c.waitForState(
		"RUNNING", operation.Header.Get("Etag"))
}

func (c *googleComputeInstance) Restart() error {

	var (
		err error

		operation *compute.Operation
	)

	logger.TraceMessage(
		"Restarting instance '%s'.", c.instance.Name)

	if operation, err = c.service.Instances.Reset(
		c.projectID,
		c.zone,
		c.instance.Name,
	).Do(); err != nil {
		return err
	}

	return c.waitForState(
		"RUNNING", operation.Header.Get("Etag"))
}

func (c *googleComputeInstance) Stop() error {

	var (
		err error

		operation *compute.Operation
	)

	logger.TraceMessage(
		"Stopping instance '%s'.", c.instance.Name)

	if operation, err = c.service.Instances.Stop(
		c.projectID,
		c.zone,
		c.instance.Name,
	).Do(); err != nil {
		return err
	}

	return c.waitForState(
		"TERMINATED", operation.Header.Get("Etag"))
}

func (c *googleComputeInstance) CanConnect(port int) bool {

	publicIP := c.PublicIP()
	if publicIP != "" {
		return canConnect(
			fmt.Sprintf("%s:%d", publicIP, port),
		)
	} else {
		return false
	}
}
