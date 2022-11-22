package cloud_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

func TestCloud(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	logger.Initialize()

	RegisterFailHandler(Fail)

	test_helpers.InitializeAWSEnvironment()
	test_helpers.InitializeAzureEnvironment()
	test_helpers.InitializeGoogleEnvironment()

	RunSpecs(t, "cloud")
}

var _ = AfterSuite(func() {

	test_helpers.CleanUpAWSTestData()
	test_helpers.CleanUpAzureTestData()
	test_helpers.CleanUpGoogleTestData()
	gexec.CleanupBuildArtifacts()
})
