package data

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/gomega"
)

func ParseConfigDocument(
	config config.Configurable,
	configDocument, configKey string,
) {

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
				if token == configKey {
					err = decoder.Decode(config)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		}
	}
}

func MarshalConfigDocumentAndValidate(
	config config.Configurable,
	configType, configKey string,
	expectedConfigDocument string,
) {

	var (
		buffer strings.Builder
	)

	// ensure defaults are bound
	_, err := config.InputForm()
	Expect(err).NotTo(HaveOccurred())

	// marshal config document
	buffer.WriteString("{\"cloud\": {\"")
	buffer.WriteString(configType)
	buffer.WriteString("\": {\"")
	buffer.WriteString(configKey)
	buffer.WriteString("\": ")

	encoder := json.NewEncoder(&buffer)
	err = encoder.Encode(config)
	Expect(err).NotTo(HaveOccurred())

	buffer.WriteString("}}}")

	// read marshalled config document back as a map of key/value pairs
	actual := make(map[string]interface{})
	err = json.Unmarshal([]byte(buffer.String()), &actual)
	Expect(err).NotTo(HaveOccurred())
	logger.TraceMessage(
		"Umarshalled saved '%s' backend config as map: %# v",
		configKey, actual)

	// read expected config document into a map of key/value pairs
	expected := make(map[string]interface{})
	err = json.Unmarshal([]byte(expectedConfigDocument), &expected)
	Expect(err).NotTo(HaveOccurred())

	// validate
	Expect(actual).To(Equal(expected))
}

func CopyConfigAndValidate(
	config config.Configurable,
	testKey, origValue, newValue string,
) {

	var (
		inputForm forms.InputForm

		// value  string
		v1, v2 *string
	)

	copy, err := config.Copy()
	Expect(err).NotTo(HaveOccurred())

	inputForm, err = config.InputForm()
	Expect(err).NotTo(HaveOccurred())

	for _, f := range inputForm.InputFields() {

		v1, err = config.GetValue(f.Name())
		Expect(err).NotTo(HaveOccurred())

		v2, err = copy.GetValue(f.Name())
		Expect(err).NotTo(HaveOccurred())

		Expect(*v2).To(Equal(*v1))
	}

	// Retrieve form again to ensure form is bound to config
	inputForm, err = config.InputForm()
	Expect(err).NotTo(HaveOccurred())

	err = inputForm.SetFieldValue(testKey, newValue)
	Expect(err).NotTo(HaveOccurred())

	// Change value in source config
	v1, err = config.GetValue(testKey)
	Expect(err).NotTo(HaveOccurred())
	Expect(*v1).To(Equal(newValue))

	// Validate change does not affect copy
	v2, err = copy.GetValue(testKey)
	Expect(err).NotTo(HaveOccurred())
	Expect(*v2).To(Equal(origValue))
}
