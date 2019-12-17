package provider_test

import (
	"bytes"
	"io"
	"os"

	"github.com/mevansam/gocloud/provider"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cloud Provider Tests", func() {

	Context("cloud provider templates", func() {

		It("validates the available cloud provider templates", func() {

			providerTemplates, err := provider.NewCloudProviderTemplates()
			Expect(err).NotTo(HaveOccurred())

			for name, template := range providerTemplates {
				Expect(name).To(Equal(template.Name()))
			}
		})
	})
})

func testConfigReferenceOutput(cloudProvider provider.CloudProvider, expected string) {

	var (
		err error

		origStdout, stdOutReader *os.File
	)

	// pipe output to be written to by form output
	origStdout = os.Stdout
	stdOutReader, os.Stdout, err = os.Pipe()
	Expect(err).ToNot(HaveOccurred())

	defer func() {
		stdOutReader.Close()
		os.Stdout = origStdout
	}()

	// channel to signal when getting form input is done
	out := make(chan string)

	go func() {

		var (
			output    bytes.Buffer
			inputForm forms.InputForm
		)

		defer GinkgoRecover()
		defer func() {
			// signal end
			out <- output.String()
		}()

		inputForm, err = cloudProvider.InputForm()
		Expect(err).NotTo(HaveOccurred())

		tf, err := ux.NewTextForm(
			"Cloud Provider Configuration",
			"CONFIGURATION DATA INPUT REFERENCE",
			inputForm)
		Expect(err).NotTo(HaveOccurred())
		tf.ShowInputReference(false, 0, 2, 80)

		// close piped output
		os.Stdout.Close()
		io.Copy(&output, stdOutReader)
	}()

	// wait until signal is received

	output := <-out
	logger.DebugMessage("\n%s\n", output)
	Expect(output).To(Equal(expected))
}
