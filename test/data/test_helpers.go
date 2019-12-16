package data

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/mevansam/goforms/config"

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

func WriteConfigDocument(
	config config.Configurable,
	configType, configKey string,
	buffer *strings.Builder,
) {

	buffer.WriteString("{\"cloud\": {\"")
	buffer.WriteString(configType)
	buffer.WriteString("\": {\"")
	buffer.WriteString(configKey)
	buffer.WriteString("\": ")
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(config)
	Expect(err).NotTo(HaveOccurred())
	buffer.WriteString("}}}")
}
