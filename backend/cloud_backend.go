package backend

import (
	"fmt"

	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
)

type CloudBackend interface {
	config.Configurable
}

type s3Backend struct {
	name string
}

type gcsBackend struct {
	name string
}

type localBackend struct {
}

func NewCloudBackend(name string) (CloudBackend, error) {

	switch name {
	case "":
		return &localBackend{}, nil

	case "s3":
		return &s3Backend{
			name: name,
		}, nil

	case "aws":
		return &gcsBackend{
			name: name,
		}, nil

	default:
		return nil,
			fmt.Errorf("backend '%s' is currently not handled by cloud builder", name)
	}
}

/**
 * S3 Backend
 */

func (b *s3Backend) Name() string {
	return b.name
}

func (b *s3Backend) Description() string {
	return ""
}

func (b *s3Backend) InputForm() (forms.InputForm, error) {
	return nil, nil
}

func (b *s3Backend) GetValue(key string) (*string, error) {
	return nil, nil
}

func (b *s3Backend) Copy() (config.Configurable, error) {
	return nil, nil
}

func (b *s3Backend) IsValid() bool {
	return false
}

func (b *s3Backend) Reset() {
}

// interface: encoding/json/Unmarshaler
func (b *s3Backend) UnmarshalJSON(in []byte) error {
	return nil
}

// interface: encoding/json/Marshaler
func (b *s3Backend) MarshalJSON() ([]byte, error) {
	return []byte{'{', '}'}, nil
}

/**
 * GCS Backend
 */

func (b *gcsBackend) Name() string {
	return b.name
}

func (b *gcsBackend) Description() string {
	return ""
}

func (b *gcsBackend) InputForm() (forms.InputForm, error) {
	return nil, nil
}

func (b *gcsBackend) GetValue(key string) (*string, error) {
	return nil, nil
}

func (b *gcsBackend) Copy() (config.Configurable, error) {
	return nil, nil
}

func (b *gcsBackend) IsValid() bool {
	return false
}

func (b *gcsBackend) Reset() {
}

// interface: encoding/json/Unmarshaler
func (b *gcsBackend) UnmarshalJSON(in []byte) error {
	return nil
}

// interface: encoding/json/Marshaler
func (b *gcsBackend) MarshalJSON() ([]byte, error) {
	return []byte{'{', '}'}, nil
}

/**
 * Local Backend
 */

func (b *localBackend) Name() string {
	return "local"
}

func (b *localBackend) Description() string {
	return "Local backend persists state in a local folder"
}

func (b *localBackend) InputForm() (forms.InputForm, error) {
	return nil, nil
}

func (b *localBackend) GetValue(key string) (*string, error) {
	return nil, nil
}

func (b *localBackend) Copy() (config.Configurable, error) {
	return nil, nil
}

func (b *localBackend) IsValid() bool {
	return false
}

func (b *localBackend) Reset() {
}

// interface: encoding/json/Unmarshaler
func (b *localBackend) UnmarshalJSON(in []byte) error {
	return nil
}

// interface: encoding/json/Marshaler
func (b *localBackend) MarshalJSON() ([]byte, error) {
	return []byte{'{', '}'}, nil
}
