package cloud

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/mevansam/goutils/logger"
)

// instance states
type InstanceState int

const (
	StateRunning = InstanceState(0)
	StateStopped = InstanceState(1)
	StatePending = InstanceState(2)
	StateUnknown = InstanceState(3)
)

func (s InstanceState) String() string {
	return []string{"running", "stopped", "pending"}[s]
}

// interface for a cloud compute abstraction
type Compute interface {

	// Properties that customize the behavior of
	// the Compute API such as search filters.
	SetProperties(props interface{})

	// Retreives a compute instance.
	GetInstance(name string) (ComputeInstance, error)

	// Returns a list of all compute instances
	// within this cloud compute context
	ListInstances() ([]ComputeInstance, error)
}

type ComputeInstance interface {
	ID() string
	Name() string
	PublicIP() string

	// Returns the instance's run state
	State() (InstanceState, error)

	// Start the instance.
	Start() error

	// Restart the instance
	Restart() error

	// Stop the instance
	Stop() error

	// Tests connectivity on a
	// given TCP port accepts
	CanConnect(port int) bool
}

// interface for a cloud object store abstraction
type Storage interface {
	SetProperties(props interface{})

	// Creates a storage instance. In the case of AWS S3
	// or Google cloud storage this will be a bucket. For
	// Azure this would be a blob container.
	NewInstance(name string) (StorageInstance, error)

	// Returns a list of all storage instance within
	// this cloud storage context
	ListInstances() ([]StorageInstance, error)
}

type StorageInstance interface {
	Name() string
	Delete() error

	ListObjects(path string) ([]string, error)
	DeleteObject(path string) error

	Upload(name, contentType string, data io.Reader, size int64) error
	UploadFile(name, contentType, path string) error

	Download(name string, data io.Writer) error
	DownloadFile(name, path string) error

	// Downloads an object asynchronously by executing
	// multiple download calls asynchronously to download
	// the blob in chunks. The caller needs to handle
	// errors from each of the asynchronous calls.
	DownloadAsync(name string, data io.WriterAt) (*sync.WaitGroup, int64, []error, error)
}

// test tcp connection
func canConnect(endpoint string) bool {

	conn, err := net.DialTimeout("tcp", endpoint, time.Second)
	if err != nil {
		logger.TraceMessage(
			"Connectivity test to '%s' failed: %s",
			endpoint, err.Error(),
		)
		return false
	}

	defer conn.Close()
	return true
}

// user agent to use for HTTP(s) API requests
const httpUserAgent = `cloud-builder`
