package provider_test

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/mevansam/goutils/config"
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

func parseConfigDocument(config config.Configurable, configDocument, providerKey string) {

	jsonStream := strings.NewReader(configDocument)
	decoder := json.NewDecoder(jsonStream)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		Expect(err).NotTo(HaveOccurred())

		if decoder.More() {
			switch token.(type) {
			case string:
				if token == providerKey {
					err = decoder.Decode(config)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		}
	}
}

func writeConfigDocument(config config.Configurable, providerKey string, buffer *strings.Builder) {

	buffer.WriteString("{\"cloud\": {\"providers\": {\"")
	buffer.WriteString(providerKey)
	buffer.WriteString("\": ")
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(config)
	Expect(err).NotTo(HaveOccurred())
	buffer.WriteString("}}}")
}
