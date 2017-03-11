package gko

import (
	"fmt"
	"io"
	"io/ioutil"

	"cloud.google.com/go/storage"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/option"
	"google.golang.org/appengine"
)

var _ CloudStorage = (*CloudStorageClient)(nil)

// Bucket is cloud storage Bucket.
type Bucket string

// Name return bucket name.
func (b Bucket) Name(ctx context.Context) string {
	return fmt.Sprintf("%s_%s", appengine.AppID(ctx), b)
}

// CloudStorageFactory is cloud storage factory interface.
type CloudStorageFactory interface {
	New(context.Context) (CloudStorage, error)
}

// CloudStorageFactoryImpl is implementation of cloud storage factory.
type CloudStorageFactoryImpl struct{}

// New return cloud storage client.
func (c *CloudStorageFactoryImpl) New(ctx context.Context) (CloudStorage, error) {
	return newCloudStorageClient(ctx)
}

// CloudStorage is cloud storage interface along with reader and writer.
type CloudStorage interface {
	CloudStorageReader
	CloudStorageWriter
}

// CloudStorageReader is cloud storage reader interface.
type CloudStorageReader interface {
	DownloadFile(Bucket, string) ([]byte, error)
}

// CloudStorageWriter is cloud storage writer interface.
type CloudStorageWriter interface {
	CreateFile(Bucket, string, io.Reader) error
	DeleteFile(Bucket, string) error
	Copy(b Bucket, src, dst string) error
	Move(b Bucket, src, dst string) error
}

// CloudStorageClient is cloud storage client.
type CloudStorageClient struct {
	ctx    context.Context
	client *storage.Client
}

// newCloudStorageClient return new cloud storage client.
//
// context and bucket, filename passed to arguments.
func newCloudStorageClient(ctx context.Context) (*CloudStorageClient, error) {
	client, err := storage.NewClient(ctx, option.WithTokenSource(google.AppEngineTokenSource(ctx)))
	if err != nil {
		ErrorLog(ctx, err.Error())
		return nil, err
	}

	return &CloudStorageClient{ctx, client}, nil
}

// DownloadFile donwload from cloud storage, then return file data.
func (c *CloudStorageClient) DownloadFile(b Bucket, filename string) ([]byte, error) {
	oh := c.client.Bucket(b.Name(c.ctx)).Object(filename)
	r, err := oh.NewReader(c.ctx)
	if err != nil {
		ErrorLog(c.ctx, err.Error())
		return nil, err
	}

	file, err := ioutil.ReadAll(r)
	if err != nil {
		ErrorLog(c.ctx, err.Error())
		return nil, err
	}
	return file, nil
}

// CreateFile create file in cloud storage.
func (c *CloudStorageClient) CreateFile(b Bucket, filename string, r io.Reader) error {
	oh := c.client.Bucket(b.Name(c.ctx)).Object(filename)
	w := oh.NewWriter(c.ctx)
	defer w.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}

	if _, err := w.Write(data); err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}

	return nil
}

// DeleteFile delete file in cloud storage.
func (c *CloudStorageClient) DeleteFile(b Bucket, filename string) error {
	oh := c.client.Bucket(b.Name(c.ctx)).Object(filename)
	if err := oh.Delete(c.ctx); err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}
	return nil
}

// Copy copy file in cloud storage from src to dst.
func (c *CloudStorageClient) Copy(b Bucket, src, dst string) error {
	soh := c.client.Bucket(b.Name(c.ctx)).Object(src)
	doh := c.client.Bucket(b.Name(c.ctx)).Object(dst)
	if _, err := doh.CopierFrom(soh).Run(c.ctx); err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}
	return nil
}

// Move move file in cloud storage from src to dst.
func (c *CloudStorageClient) Move(b Bucket, src, dst string) error {
	if err := c.Copy(b, src, dst); err != nil {
		return err
	}

	if err := c.DeleteFile(b, src); err != nil {
		return err
	}

	return nil
}
