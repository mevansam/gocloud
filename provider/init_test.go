package provider_test

import (
	"testing"

	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	test_helpers "github.com/mevansam/gocloud/test/helpers"
)

func TestConfig(t *testing.T) {
	logger.Initialize()

	RegisterFailHandler(Fail)

	test_helpers.InitializeAWSEnvironment()
	test_helpers.InitializeAzureEnvironment()
	test_helpers.InitializeGoogleEnvironment()

	RunSpecs(t, "provider")
}

var _ = AfterSuite(func() {

	test_helpers.CleanUpAzureTestData()
	gexec.CleanupBuildArtifacts()
})
