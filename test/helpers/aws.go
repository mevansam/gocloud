package helpers

import (
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (

	// azure credentials from environment
	awsAccessKeyID,
	awsSecretAccessKey,
	awsDefaultRegion string
)

// read azure environment
func InitializeAWSEnvironment() {

	defer GinkgoRecover()

	// retrieve aws credentials from environment
	if awsAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID"); len(awsAccessKeyID) == 0 {
		Fail("environment variable named AWS_ACCESS_KEY_ID must be provided")
	}
	if awsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY"); len(awsSecretAccessKey) == 0 {
		Fail("environment variable named AWS_SECRET_ACCESS_KEY must be provided")
	}
	if awsDefaultRegion = os.Getenv("AWS_DEFAULT_REGION"); len(awsSecretAccessKey) == 0 {
		Fail("environment variable named AWS_DEFAULT_REGION must be provided")
	}
}

// update azure provider with environment credentials
func InitializeAWSProvider(awsProvider provider.CloudProvider) {

	var (
		err error

		inputForm forms.InputForm
	)

	inputForm, err = awsProvider.InputForm()
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("access_key", awsAccessKeyID)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("secret_key", awsSecretAccessKey)
	Expect(err).NotTo(HaveOccurred())
	err = inputForm.SetFieldValue("region", awsDefaultRegion)
	Expect(err).NotTo(HaveOccurred())
}

// cleans up any test data created in Azure account
func CleanUpAWSTestData() {

	var (
		err error

		sess           *session.Session
		describeResult *ec2.DescribeInstancesOutput
	)

	if noCleanUp := os.Getenv("AWS_NO_CLEANUP"); noCleanUp != "1" {

		sess, err = session.NewSession(&aws.Config{
			Region: aws.String(awsDefaultRegion),
			Credentials: credentials.NewStaticCredentials(
				awsAccessKeyID,
				awsSecretAccessKey,
				"",
			),
		})
		Expect(err).NotTo(HaveOccurred())
		svc := ec2.New(sess)

		describeResult, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("tag:Role"),
					Values: []*string{aws.String("Cloudbuilder-Test")},
				},
				&ec2.Filter{
					Name: aws.String("instance-state-name"),
					Values: []*string{
						aws.String("pending"),
						aws.String("running"),
						aws.String("shutting-down"),
						aws.String("stopping"),
						aws.String("stopped"),
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		if describeResult.Reservations != nil {

			for _, r := range describeResult.Reservations {

				logger.TraceMessage(
					"Terminating test instance with ID '%s'.",
					*r.Instances[0].InstanceId)

				_, err = svc.TerminateInstances(&ec2.TerminateInstancesInput{
					InstanceIds: []*string{r.Instances[0].InstanceId},
				})
				Expect(err).NotTo(HaveOccurred())
			}
		}
	}
}

// creates aws instances for testing
func AWSDeployTestInstances(name string, numInstances int) map[string]string {

	var (
		err error
		wg  sync.WaitGroup

		sess *session.Session

		instanceIPs map[string]string
	)

	sess, err = session.NewSession(&aws.Config{
		Region: aws.String(awsDefaultRegion),
		Credentials: credentials.NewStaticCredentials(
			awsAccessKeyID,
			awsSecretAccessKey,
			"",
		),
	})
	Expect(err).NotTo(HaveOccurred())
	svc := ec2.New(sess)

	instanceIPs = make(map[string]string)

	wg.Add(numInstances)
	for i := 0; i < numInstances; i++ {

		go func(i int) {
			defer wg.Done()
			defer GinkgoRecover()

			var (
				describeResult *ec2.DescribeInstancesOutput
				runResult      *ec2.Reservation
				instance       *ec2.Instance
			)
			vmName := fmt.Sprintf("%s-%d", name, i)

			describeResult, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
				Filters: []*ec2.Filter{
					&ec2.Filter{
						Name:   aws.String("tag:Name"),
						Values: []*string{aws.String(vmName)},
					},
					&ec2.Filter{
						Name: aws.String("instance-state-name"),
						Values: []*string{
							aws.String("pending"),
							aws.String("running"),
							aws.String("shutting-down"),
							aws.String("stopping"),
							aws.String("stopped"),
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			if describeResult.Reservations == nil || len(describeResult.Reservations) == 0 {

				logger.TraceMessage(
					"Creating instance with name '%s'.",
					vmName)

				runResult, err = svc.RunInstances(&ec2.RunInstancesInput{
					ImageId:      aws.String("ami-00068cd7555f543d5"),
					InstanceType: aws.String("t3.nano"),
					MinCount:     aws.Int64(1),
					MaxCount:     aws.Int64(1),
				})
				Expect(err).NotTo(HaveOccurred())
				instance = runResult.Instances[0]

				_, err = svc.CreateTags(&ec2.CreateTagsInput{
					Resources: []*string{instance.InstanceId},
					Tags: []*ec2.Tag{
						{
							Key:   aws.String("Name"),
							Value: aws.String(vmName),
						},
						{
							Key:   aws.String("Role"),
							Value: aws.String("Cloudbuilder-Test"),
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				instance = awsWaitUntilInstanceRunning(svc, instance.InstanceId)

			} else {

				instance = (*describeResult.Reservations[0]).Instances[0]
				switch *instance.State.Name {
				case "running":
					// Continue as normal
				case "stopped":
					logger.TraceMessage(
						"Starting test instance with ID '%s'.",
						*instance.InstanceId)

					_, err = svc.StartInstances(&ec2.StartInstancesInput{
						InstanceIds: []*string{instance.InstanceId},
					})
					instance = awsWaitUntilInstanceRunning(svc, instance.InstanceId)
				default:
					Fail(
						fmt.Sprintf(
							"Found instance '%s' but it is not in a running or stopped state: %s",
							vmName, *instance.State.Name,
						),
					)
				}
			}

			logger.TraceMessage(
				"Using instance: ID - %s, name - %s",
				*instance.InstanceId, vmName)

			instanceIPs[vmName] = *instance.PublicIpAddress
		}(i)
	}
	wg.Wait()

	return instanceIPs
}

func awsWaitUntilInstanceRunning(svc *ec2.EC2, instanceID *string) *ec2.Instance {

	var (
		err error

		describeResult *ec2.DescribeInstancesOutput
	)

	err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-id"),
				Values: []*string{instanceID},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())

	describeResult, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-id"),
				Values: []*string{instanceID},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(len((*describeResult.Reservations[0]).Instances)).To(Equal(1))

	return (*describeResult.Reservations[0]).Instances[0]
}

func AWSInstanceState(id string) string {

	var (
		err error

		sess           *session.Session
		describeResult *ec2.DescribeInstancesOutput
	)

	sess, err = session.NewSession(&aws.Config{
		Region: aws.String(awsDefaultRegion),
		Credentials: credentials.NewStaticCredentials(
			awsAccessKeyID,
			awsSecretAccessKey,
			"",
		),
	})
	Expect(err).NotTo(HaveOccurred())
	svc := ec2.New(sess)

	describeResult, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	Expect(err).NotTo(HaveOccurred())

	return *(*describeResult.Reservations[0]).Instances[0].State.Name
}
