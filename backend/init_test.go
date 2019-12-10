package backend_test

import (
	"testing"

	"github.com/mevansam/goutils/logger"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	logger.Initialize()

	RegisterFailHandler(Fail)
	RunSpecs(t, "backend")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
