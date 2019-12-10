package cloud

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/mevansam/goutils/logger"
)

type GoogleStorageProperties struct {
	Region string

	// size of block when uploading
	// blocks to a blob concurrently
	BlockSize int
}

type googleStorage struct {
	client    *storage.Client
	projectID string

	ctx context.Context

	props GoogleStorageProperties
}

type googleStorageInstance struct {
	client *storage.Client

	projectID,
	name string

	ctx context.Context

	props *GoogleStorageProperties
}

func NewGoogleStorage(
	ctx context.Context,
	client *storage.Client,
	projectID,
	region string,
) (Storage, error) {

	return &googleStorage{
		client:    client,
		projectID: projectID,

		ctx: ctx,

		props: GoogleStorageProperties{
			Region:    region,
			BlockSize: 5 * 1024 * 1024, // 5MB
		},
	}, nil
}

// interface: cloud/Storage implementation

func (s *googleStorage) SetProperties(props interface{}) {

	p := props.(GoogleStorageProperties)
	if len(p.Region) > 0 {
		s.props.Region = p.Region
	}
}

func (s *googleStorage) NewInstance(name string) (StorageInstance, error) {

	var (
		err error
	)

	bucket := s.client.Bucket(name)
	if _, err = bucket.Attrs(s.ctx); err != nil {
		if err.Error() == "storage: bucket doesn't exist" {

			logger.TraceMessage(
				"Bucket '%s' was not found so creating it.",
				name)

			if err := bucket.Create(s.ctx, s.projectID, &storage.BucketAttrs{
				Location: s.props.Region,
			}); err != nil {
				return nil, err
			}

		} else {
			return nil, err
		}
	}

	return &googleStorageInstance{
		client: s.client,

		projectID: s.projectID,
		name:      name,

		ctx: s.ctx,

		props: &s.props,
	}, nil
}

func (s *googleStorage) ListInstances() ([]StorageInstance, error) {

	var (
		err error

		attrs *storage.BucketAttrs
	)
	instances := []StorageInstance{}
	location := strings.ToUpper(s.props.Region)

	i := s.client.Buckets(s.ctx, s.projectID)
	for {
		attrs, err = i.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if attrs.Location == location {
			instances = append(instances, &googleStorageInstance{
				client: s.client,

				projectID: s.projectID,
				name:      attrs.Name,

				ctx: s.ctx,

				props: &s.props,
			})
		}
	}
	return instances, nil
}

// interface: cloud/StorageInstance implementation

func (s *googleStorageInstance) Name() string {
	return s.name
}

func (s *googleStorageInstance) Delete() error {
	return s.client.Bucket(s.name).Delete(s.ctx)
}

func (s *googleStorageInstance) ListObjects(path string) ([]string, error) {

	var (
		err error

		attrs *storage.ObjectAttrs
	)
	objects := []string{}

	i := s.client.Bucket(s.name).Objects(s.ctx, &storage.Query{
		Prefix: path,
	})
	for {
		attrs, err = i.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		objects = append(objects, attrs.Name)
	}
	return objects, nil
}

func (s *googleStorageInstance) DeleteObject(name string) error {
	return s.client.Bucket(s.name).Object(name).Delete(s.ctx)
}

func (s *googleStorageInstance) Upload(name, contentType string, data io.Reader, size int64) error {

	var (
		err error

		writer *storage.Writer
	)
	logger.TraceMessage(
		"Uploading object with name '%s' of size %d to bucket '%s'.",
		name, size, s.name)

	writer = s.client.Bucket(s.name).Object(name).NewWriter(s.ctx)
	writer.ChunkSize = s.props.BlockSize
	writer.ContentType = contentType

	if _, err = io.CopyBuffer(
		writer,
		data,
		make([]byte, s.props.BlockSize),
	); err != nil {
		return err
	}
	return writer.Close()
}

func (s *googleStorageInstance) UploadFile(name, contentType, path string) error {

	var (
		err error

		file     *os.File
		fileInfo os.FileInfo
	)

	if file, err = os.Open(path); err != nil {
		return err
	}
	defer file.Close()

	if fileInfo, err = file.Stat(); err != nil {
		return err
	}
	return s.Upload(name, contentType, file, fileInfo.Size())
}

func (s *googleStorageInstance) Download(name string, data io.Writer) error {

	var (
		err error

		reader *storage.Reader
	)
	logger.TraceMessage(
		"Downloading object with name '%s' from bucket '%s'.",
		name, s.name)

	if reader, err = s.client.Bucket(s.name).
		Object(name).NewReader(s.ctx); err != nil {
		return err
	}

	_, err = io.CopyBuffer(
		data,
		reader,
		make([]byte, s.props.BlockSize),
	)
	return err
}

func (s *googleStorageInstance) DownloadFile(name, path string) error {

	var (
		err error

		file *os.File

		wg     *sync.WaitGroup
		size   int64
		errors []error
	)

	if file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644); err != nil {
		return err
	}
	defer file.Close()

	wg, size, errors, err = s.DownloadAsync(name, file)
	wg.Wait()

	if err != nil {
		var errMsg strings.Builder
		errMsg.WriteString(err.Error())

		for i := 0; i < len(errors); i++ {
			if errors[i] != nil {
				errMsg.WriteString(
					fmt.Sprintf("; Uploading block %d failed: %s", i, errors[i].Error()),
				)
			}
		}
		return fmt.Errorf(errMsg.String())
	}

	err = file.Truncate(size)
	return err
}

func (s *googleStorageInstance) DownloadAsync(name string, data io.WriterAt) (*sync.WaitGroup, int64, []error, error) {

	var (
		err error
		wg  sync.WaitGroup

		attrs *storage.ObjectAttrs
	)

	// get size of blob to download
	object := s.client.Bucket(s.name).Object(name)
	if attrs, err = object.Attrs(s.ctx); err != nil {
		return &wg, 0, nil, err
	}
	size := attrs.Size

	logger.TraceMessage(
		"Downloading object with name '%s' of size %d from bucket '%s'.",
		name, size, s.name)

	numBlocks := int(size / int64(s.props.BlockSize))
	partialBlockSize := int(size % int64(s.props.BlockSize))
	if partialBlockSize > 0 {
		numBlocks++
	}

	hasErrors := false
	errors := make([]error, numBlocks)

	wg.Add(numBlocks)
	for i := 0; i < numBlocks; i++ {

		go func(blockNum int) {
			defer wg.Done()

			var (
				reader *storage.Reader

				n   int
				b   []byte
				eof bool
			)

			logger.TraceMessage(
				"Downloading block %d of object with name '%s' in bucket '%s'.",
				blockNum, name, s.name)

			offset := int64(blockNum) * int64(s.props.BlockSize)
			length := int64(s.props.BlockSize)
			if offset+length > size {
				length = size - offset
			}

			if reader, err = s.client.Bucket(s.name).
				Object(name).NewRangeReader(s.ctx, offset, length); err != nil {

				errors[blockNum] = err
				hasErrors = true
				return
			}

			b = make([]byte, s.props.BlockSize)
			eof = false
			for !eof {
				if n, err = reader.Read(b); err != nil {
					if err != io.EOF {
						errors[blockNum] = err
						hasErrors = true
						return
					} else {
						eof = true
					}
				}
				if n > 0 {
					if _, err = data.WriteAt(b[0:n], offset); err != nil {
						errors[blockNum] = err
						hasErrors = true
						return
					}
					offset = offset + int64(n)
				}
			}
		}(i)
	}

	if hasErrors {
		return &wg, size, errors,
			fmt.Errorf("failed to download object '%s' from bucket '%s'", name, s.name)
	} else {
		return &wg, size, nil, nil
	}
}
