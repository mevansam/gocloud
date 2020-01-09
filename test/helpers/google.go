package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"

	"google.golang.org/api/option"

	compute "google.golang.org/api/compute/v1"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (

	// google credentials from environment
	googleCredentialsFile,
	googleProject,
	googleRegion,
	googleZone string

	googleCredentialsJson []byte
	googleCredentials     map[string]interface{}

	googleRegions []string
)

// read google environment
func InitializeGoogleEnvironment() {

	var (
		err error
	)

	defer GinkgoRecover()

	if googleCredentialsFile = os.Getenv("GOOGLE_CREDENTIALS"); len(googleCredentialsFile) == 0 {
		Fail("environment variable named GOOGLE_CREDENTIALS with the path to the Google service credentials JSON must be provided")
	}
	googleCredentialsJson, err = ioutil.ReadFile(googleCredentialsFile)
	Expect(err).NotTo(HaveOccurred())

	googleCredentials = make(map[string]interface{})
	err = json.Unmarshal(googleCredentialsJson, &googleCredentials)
	Expect(err).NotTo(HaveOccurred())

	if googleProject = os.Getenv("GOOGLE_PROJECT"); len(googleProject) == 0 {
		googleProject = googleCredentials["project_id"].(string)
	}
	if googleRegion = os.Getenv("GOOGLE_REGION"); len(googleRegion) == 0 {
		Fail("environment variable named GOOGLE_REGION must be provided")
	}
	if googleZone = os.Getenv("GOOGLE_ZONE"); len(googleZone) == 0 {
		Fail("environment variable named GOOGLE_ZONE must be provided")
	}
}

// cleans up any test data created in Google account
func CleanUpGoogleTestData() {

	var (
		err error

		computeService *compute.Service
		instanceList   *compute.InstanceList

		call *compute.InstancesListCall
	)

	if noCleanUp := os.Getenv("GOOGLE_NO_CLEANUP"); noCleanUp != "1" {

		computeService, err = compute.NewService(
			context.Background(), option.WithCredentialsJSON(googleCredentialsJson))
		Expect(err).NotTo(HaveOccurred())

		call = computeService.Instances.List(googleProject, googleZone)
		call.Filter("labels.role=cloudbuilder-test")

		instanceList, err = call.Do()
		Expect(err).NotTo(HaveOccurred())

		if instanceList.Items != nil {
			for _, instance := range instanceList.Items {
				_, err = computeService.Instances.Delete(googleProject, googleZone, instance.Name).Do()
				Expect(err).NotTo(HaveOccurred())
			}
		}
	}
}

// update google provider with environment credentials
func InitializeGoogleProvider(googleProvider provider.CloudProvider) {

	var (
		err error

		inputForm forms.InputForm
	)

	inputForm, err = googleProvider.InputForm()
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("credentials", googleCredentialsFile)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("project", googleProject)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("region", googleRegion)
	Expect(err).NotTo(HaveOccurred())
}

// load azure regions/locations via API calls
func GoogleGetRegions() []string {

	if googleRegions == nil {
		googleRegions = []string{}

		// retrieve regions from Google API - this is an authenticated call
		// so we validate the response from Google API with hardcoded list
		// returned by provide making the test fail if new regions are found
		// indicating that provide code needs to be updated.
		ctx := context.Background()

		computeService, err := compute.NewService(ctx, option.WithCredentialsJSON(googleCredentialsJson))
		Expect(err).NotTo(HaveOccurred())

		logger.DebugMessage("\nGoogle regions retrieved from API call:")

		err = computeService.Regions.List(googleProject).
			Pages(ctx,
				func(page *compute.RegionList) error {
					for _, region := range page.Items {
						logger.DebugMessage("  * %s", region.Name)
						googleRegions = append(googleRegions, region.Name)
					}
					return nil
				})
		Expect(err).NotTo(HaveOccurred())
		sort.Strings(googleRegions)
	}
	return googleRegions
}

// creates google instances for testing
func GoogleDeployTestInstances(name string, numInstances int) map[string]*compute.Instance {

	var (
		err error
		wg  sync.WaitGroup
	)

	instances := make(map[string]*compute.Instance)

	ctx := context.Background()
	computeService, err := compute.NewService(ctx, option.WithCredentialsJSON(googleCredentialsJson))
	Expect(err).NotTo(HaveOccurred())

	prefix := "https://www.googleapis.com/compute/v1/projects/" + googleProject
	imageURL := "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20191113"

	wg.Add(numInstances)
	for i := 0; i < numInstances; i++ {

		go func(i int) {
			defer wg.Done()
			defer GinkgoRecover()

			var (
				instance  *compute.Instance
				operation *compute.Operation
			)

			vmName := fmt.Sprintf("%s-%d", name, i)
			if instance, err = computeService.Instances.Get(
				googleProject,
				googleZone,
				vmName,
			).Do(); err != nil {

				logger.TraceMessage(
					"Creating instance with name '%s'.",
					vmName)

				instance = &compute.Instance{
					Name:        vmName,
					Description: "compute sample instance",
					MachineType: prefix + "/zones/" + googleZone + "/machineTypes/n1-standard-1",
					Disks: []*compute.AttachedDisk{
						{
							AutoDelete: true,
							Boot:       true,
							Type:       "PERSISTENT",
							InitializeParams: &compute.AttachedDiskInitializeParams{
								DiskName:    vmName + "-root-disk",
								SourceImage: imageURL,
							},
						},
					},
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							AccessConfigs: []*compute.AccessConfig{
								{
									Type: "ONE_TO_ONE_NAT",
									Name: "External NAT",
								},
							},
							Network: prefix + "/global/networks/default",
						},
					},
					ServiceAccounts: []*compute.ServiceAccount{
						{
							Email: "default",
							Scopes: []string{
								compute.DevstorageFullControlScope,
								compute.ComputeScope,
							},
						},
					},
					Labels: map[string]string{
						"role": "cloudbuilder-test",
					},
				}

				operation, err = computeService.Instances.Insert(googleProject, googleZone, instance).Do()
				Expect(err).NotTo(HaveOccurred())
				etag := operation.Header.Get("Etag")

				for instance.Status != "RUNNING" {
					instance, err = computeService.Instances.Get(googleProject, googleZone, vmName).IfNoneMatch(etag).Do()
					Expect(err).NotTo(HaveOccurred())
				}
			}

			logger.TraceMessage(
				"Using instance: ID - %s, name - %s",
				instance.Id, instance.Name)

			instances[vmName] = instance
		}(i)
	}
	wg.Wait()

	return instances
}

func GoogleInstanceState(name string) string {

	ctx := context.Background()
	computeService, err := compute.NewService(ctx, option.WithCredentialsJSON(googleCredentialsJson))
	Expect(err).NotTo(HaveOccurred())

	instance, err := computeService.Instances.Get(googleProject, googleZone, name).Do()
	Expect(err).NotTo(HaveOccurred())
	return instance.Status
}
