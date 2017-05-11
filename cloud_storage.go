package gko

import (
	"io"
	"io/ioutil"

	"cloud.google.com/go/storage"

	"golang.org/x/net/context"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

var (
	_ CloudStorageFactory = (*cloudStorageFactoryImpl)(nil)
	_ CloudStorage        = (*cloudStorageClient)(nil)
)

var cloudStorageFactory CloudStorageFactory

// CloudStorageFactory is cloud storage factory interface.
type CloudStorageFactory interface {
	New(context.Context, oauth2.TokenSource) (CloudStorage, error)
}

// cloudStorageFactoryImpl is implementation of cloud storage factory.
type cloudStorageFactoryImpl struct{}

// CloudStorage is cloud storage interface along with reader and writer.
type CloudStorage interface {
	CloudStorageReader
	CloudStorageWriter
}

// CloudStorageReader is cloud storage reader interface.
type CloudStorageReader interface {
	DownloadFile(string, string) ([]byte, error)
}

// CloudStorageWriter is cloud storage writer interface.
type CloudStorageWriter interface {
	CreateFile(string, string, io.Reader) error
	DeleteFile(string, string) error
	Copy(srcBucket string, dstBucket string, srcFile string, dstFile string) error
	Move(srcBucket string, dstBucket string, srcFile string, dstFile string) error
}

// cloudStorageClient is cloud storage client.
type cloudStorageClient struct {
	ctx    context.Context
	client *storage.Client
}

// GetCloudStorageFactory return cloud storage factory.
func GetCloudStorageFactory() CloudStorageFactory {
	if cloudStorageFactory == nil {
		cloudStorageFactory = &cloudStorageFactoryImpl{}
	}
	return cloudStorageFactory
}

// New return cloud storage client.
//
// If ts is specified, replace default google token to specified token source.
func (c *cloudStorageFactoryImpl) New(ctx context.Context, ts oauth2.TokenSource) (CloudStorage, error) {
	var opts []option.ClientOption
	if ts != nil {
		opts = append(opts, option.WithTokenSource(ts))
	}
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &cloudStorageClient{ctx, client}, nil
}

// DownloadFile donwload from cloud storage, then return file data.
func (c *cloudStorageClient) DownloadFile(bucket string, filename string) ([]byte, error) {
	r, err := c.client.Bucket(bucket).Object(filename).NewReader(c.ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	file, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// CreateFile create file in cloud storage.
func (c *cloudStorageClient) CreateFile(bucket string, filename string, r io.Reader) error {
	w := c.client.Bucket(bucket).Object(filename).NewWriter(c.ctx)
	defer w.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}

// DeleteFile delete file in cloud storage.
func (c *cloudStorageClient) DeleteFile(bucket string, filename string) error {
	oh := c.client.Bucket(bucket).Object(filename)
	if err := oh.Delete(c.ctx); err != nil {
		return err
	}
	return nil
}

// Copy copy file in cloud storage from src to dst.
func (c *cloudStorageClient) Copy(srcBucket string, dstBucket string, srcFile string, dstFile string) error {
	soh := c.client.Bucket(srcBucket).Object(srcFile)
	doh := c.client.Bucket(dstBucket).Object(dstFile)
	if _, err := doh.CopierFrom(soh).Run(c.ctx); err != nil {
		return err
	}
	return nil
}

// Move move file in cloud storage from src to dst.
func (c *cloudStorageClient) Move(srcBucket string, dstBucket string, srcFile string, dstFile string) error {
	if err := c.Copy(srcBucket, dstBucket, srcFile, dstFile); err != nil {
		return err
	}

	if err := c.DeleteFile(srcBucket, srcFile); err != nil {
		return err
	}

	return nil
}
