package cloud

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

type AWSStorageProperties struct {
	Region string

	// size of block when uploading
	// blocks to a blob concurrently
	BlockSize int64

	// Concurrency
	UploadConcurrency   int
	DownloadConcurrency int
}

type awsStorage struct {
	session *session.Session

	props AWSStorageProperties
}

type awsStorageInstance struct {
	name    string
	session *session.Session

	props *AWSStorageProperties
}

func NewAWSStorage(
	session *session.Session,
	region string,
) (Storage, error) {

	return &awsStorage{
		session: session,

		props: AWSStorageProperties{
			Region:              region,
			BlockSize:           s3manager.DefaultUploadPartSize,
			UploadConcurrency:   s3manager.DefaultUploadConcurrency,
			DownloadConcurrency: s3manager.DefaultDownloadConcurrency,
		},
	}, nil
}

// interface: cloud/Storage implementation

func (s *awsStorage) SetProperties(props interface{}) {

	p := props.(AWSStorageProperties)
	if len(p.Region) > 0 {
		s.props.Region = p.Region
	}
	if p.BlockSize > 0 {
		s.props.BlockSize = p.BlockSize
	}
	if p.UploadConcurrency > 0 {
		s.props.UploadConcurrency = p.UploadConcurrency
	}
	if p.DownloadConcurrency > 0 {
		s.props.DownloadConcurrency = p.DownloadConcurrency
	}
}

func (s *awsStorage) NewInstance(name string) (StorageInstance, error) {

	var (
		err error

		bucketLocationResult *s3.GetBucketLocationOutput
		bucketConfiguration  *s3.CreateBucketConfiguration
	)
	svc := s3.New(s.session)

	if s.props.Region != "us-east-1" {
		// bucket location is required only if bucket is being
		// created in a region other than 'us-east-1'.
		bucketConfiguration = &s3.CreateBucketConfiguration{
			LocationConstraint: &s.props.Region,
		}
	}

	if bucketLocationResult, err = svc.GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: aws.String(name),
	}); err != nil {

		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchBucket {
				// create bucket
				logger.TraceMessage(
					"Creating bucket '%s' at location '%s' with private access.",
					name, s.props.Region)

				if _, err = svc.CreateBucket(&s3.CreateBucketInput{
					Bucket: aws.String(name),

					ACL:                        aws.String("private"),
					CreateBucketConfiguration:  bucketConfiguration,
					ObjectLockEnabledForBucket: aws.Bool(true),
				}); err != nil {
					return nil, err
				}
				if err = svc.WaitUntilBucketExists(&s3.HeadBucketInput{
					Bucket: aws.String(name),
				}); err != nil {
					return nil, err
				}
				if _, err = svc.PutPublicAccessBlock(&s3.PutPublicAccessBlockInput{
					Bucket: aws.String(name),

					PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
						BlockPublicAcls:       aws.Bool(true),
						BlockPublicPolicy:     aws.Bool(true),
						IgnorePublicAcls:      aws.Bool(true),
						RestrictPublicBuckets: aws.Bool(true),
					},
				}); err != nil {
					return nil, err
				}

			} else {
				return nil, err
			}
		}
	} else {
		logger.DebugMessage(
			"Retreived bucket '%s' with the following location information: %# v",
			name, bucketLocationResult)
	}

	return &awsStorageInstance{
		name:    name,
		session: s.session,

		props: &s.props,
	}, nil
}

func (s *awsStorage) ListInstances() ([]StorageInstance, error) {

	var (
		err error

		buckerListResult *s3.ListBucketsOutput
	)
	svc := s3.New(s.session)

	if buckerListResult, err = svc.ListBuckets(nil); err != nil {
		return nil, err
	}

	instances := []StorageInstance{}
	for _, b := range buckerListResult.Buckets {

		instances = append(instances, &awsStorageInstance{
			name:    *b.Name,
			session: s.session,
		})
	}

	return instances, nil
}

// interface: cloud/StorageInstance implementation

func (s *awsStorageInstance) Name() string {
	return s.name
}

func (s *awsStorageInstance) Delete() error {

	var (
		err error
	)
	svc := s3.New(s.session)

	if _, err = svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(s.name),
	}); err != nil {
		return err
	}
	err = svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: aws.String(s.name),
	})
	return err
}

func (s *awsStorageInstance) ListObjects(path string) ([]string, error) {

	var (
		err error

		contToken *string
		resp      *s3.ListObjectsV2Output
	)
	svc := s3.New(s.session)

	objectList := []string{}
	for {
		if resp, err = svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(s.name),
			Prefix: aws.String(path),

			ContinuationToken: contToken,
		}); err != nil {
			return nil, err
		}
		logger.TraceMessage(
			"Retrieved list of objects in bucket '%s' filtered by path '%s': %# v",
			s.name, path, resp.Contents)

		for _, item := range resp.Contents {
			objectList = append(objectList, *item.Key)
		}
		contToken = resp.ContinuationToken
		if contToken == nil || len(*contToken) == 0 {
			break
		}
	}

	return objectList, nil
}

func (s *awsStorageInstance) DeleteObject(name string) error {

	var (
		err error

		versions *s3.ListObjectVersionsOutput
	)
	svc := s3.New(s.session)

	if versions, err = svc.ListObjectVersions(&s3.ListObjectVersionsInput{
		Bucket: aws.String(s.name),
		Prefix: aws.String(name),
	}); err != nil {
		return err
	}
	if versions.Versions != nil {
		// delete all versions of object
		objectsToDelete := make([]*s3.ObjectIdentifier, len(versions.Versions))
		for i, v := range versions.Versions {
			logger.TraceMessage(
				"Deleting version '%s' of object '%s' in bucket '%s'.",
				*v.Key, name, s.name)

			objectsToDelete[i] = &s3.ObjectIdentifier{
				Key:       v.Key,
				VersionId: v.VersionId,
			}
		}
		if _, err = svc.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: aws.String(s.name),
			Delete: &s3.Delete{
				Objects: objectsToDelete,
				Quiet:   aws.Bool(true),
			},
		}); err != nil {
			return err
		}

	} else {
		logger.TraceMessage(
			"Deleting object '%s' in bucket '%s'.",
			name, s.name)

		if _, err = svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(s.name),
			Key:    aws.String(name),
		}); err != nil {
			return err
		}
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(s.name),
		Key:    aws.String(name),
	})
	return err
}

func (s *awsStorageInstance) Upload(name, contentType string, data io.Reader, size int64) error {

	var (
		err error
	)
	uploader := s3manager.NewUploader(s.session, func(u *s3manager.Uploader) {
		if s.props.BlockSize > s3manager.DefaultUploadPartSize {
			u.PartSize = s.props.BlockSize
		}
		u.Concurrency = s.props.UploadConcurrency
	})
	logger.TraceMessage(
		"Uploading object with name '%s' of size %d to bucket '%s'.",
		name, size, s.name)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.name),
		Key:    aws.String(name),

		ContentType: aws.String(contentType),
		Body:        data,
	})
	return err
}

func (s *awsStorageInstance) UploadFile(name, contentType, path string) error {

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

func (s *awsStorageInstance) Download(name string, data io.Writer) error {

	var (
		err error
	)
	downloader := s3manager.NewDownloader(s.session, func(d *s3manager.Downloader) {
		if s.props.BlockSize > s3manager.DefaultDownloadPartSize {
			d.PartSize = s.props.BlockSize
		}
		d.Concurrency = s.props.DownloadConcurrency
	})
	logger.TraceMessage(
		"Downloading object with name '%s' from bucket '%s'.",
		name, s.name)

	output := utils.NewWriteAtBuffer(data)
	if _, err = downloader.Download(output,
		&s3.GetObjectInput{
			Bucket: aws.String(s.name),
			Key:    aws.String(name),
		}); err != nil {
		return err
	}

	err = output.Close()
	return err
}

func (s *awsStorageInstance) DownloadFile(name, path string) error {

	var (
		err error

		file *os.File

		wg     *sync.WaitGroup
		errors []error
	)

	if file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644); err != nil {
		return err
	}
	defer file.Close()

	logger.TraceMessage(
		"Downloading object with name '%s' from bucket '%s' to path '%s'.",
		name, s.name, path)

	wg, _, errors, err = s.DownloadAsync(name, file)
	wg.Wait()

	if err != nil {
		var errMsg strings.Builder
		errMsg.WriteString(err.Error())

		if errors[0] != nil {
			errMsg.WriteString(
				fmt.Sprintf(": %s", errors[0].Error()),
			)
		}
		return fmt.Errorf(errMsg.String())
	}

	return err
}

func (s *awsStorageInstance) DownloadAsync(name string, data io.WriterAt) (*sync.WaitGroup, int64, []error, error) {

	var (
		err  error
		wg   sync.WaitGroup
		size int64
	)
	downloader := s3manager.NewDownloader(s.session, func(d *s3manager.Downloader) {
		if s.props.BlockSize > s3manager.DefaultDownloadPartSize {
			d.PartSize = s.props.BlockSize
		}
		d.Concurrency = s.props.DownloadConcurrency
	})

	// s3 API downloads the object using asynchronous
	// GET calls. so we simply invoke the s3 manager's
	// download function asynchronously
	wg.Add(1)

	hasErrors := false
	errors := make([]error, 1)

	go func() {
		defer wg.Done()

		if size, err = downloader.Download(data,
			&s3.GetObjectInput{
				Bucket: aws.String(s.name),
				Key:    aws.String(name),
			}); err != nil {
			hasErrors = true
			errors[0] = err
		}
	}()

	if hasErrors {
		return &wg, size, errors,
			fmt.Errorf("failed to download object '%s' from bucket '%s'", name, s.name)
	} else {
		return &wg, size, nil, nil
	}
}
