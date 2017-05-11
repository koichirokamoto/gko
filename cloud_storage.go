package gko

import (
	"fmt"
	"io"
	"io/ioutil"

	"cloud.google.com/go/storage"

	"golang.org/x/net/context"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/appengine"
)

var (
	_ CloudStorageFactory = (*cloudStorageFactoryImpl)(nil)
	_ CloudStorage        = (*cloudStorageClient)(nil)
)

var cloudStorageFactory CloudStorageFactory

// Bucket is cloud storage Bucket.
type Bucket string

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
	DownloadFile(Bucket, string) ([]byte, error)
}

// CloudStorageWriter is cloud storage writer interface.
type CloudStorageWriter interface {
	CreateFile(Bucket, string, io.Reader) error
	DeleteFile(Bucket, string) error
	Copy(b Bucket, src, dst string) error
	Move(b Bucket, src, dst string) error
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

// Name return bucket name.
func (b Bucket) Name(ctx context.Context) string {
	return fmt.Sprintf("%s_%s", appengine.AppID(ctx), b)
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
func (c *cloudStorageClient) DownloadFile(b Bucket, filename string) ([]byte, error) {
	oh := c.client.Bucket(b.Name(c.ctx)).Object(filename)
	r, err := oh.NewReader(c.ctx)
	if err != nil {
		return nil, err
	}

	file, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// CreateFile create file in cloud storage.
func (c *cloudStorageClient) CreateFile(b Bucket, filename string, r io.Reader) error {
	oh := c.client.Bucket(b.Name(c.ctx)).Object(filename)
	w := oh.NewWriter(c.ctx)
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
func (c *cloudStorageClient) DeleteFile(b Bucket, filename string) error {
	oh := c.client.Bucket(b.Name(c.ctx)).Object(filename)
	if err := oh.Delete(c.ctx); err != nil {
		return err
	}
	return nil
}

// Copy copy file in cloud storage from src to dst.
func (c *cloudStorageClient) Copy(b Bucket, src, dst string) error {
	bh := c.client.Bucket(b.Name(c.ctx))
	soh := bh.Object(src)
	doh := bh.Object(dst)
	if _, err := doh.CopierFrom(soh).Run(c.ctx); err != nil {
		return err
	}
	return nil
}

// Move move file in cloud storage from src to dst.
func (c *cloudStorageClient) Move(b Bucket, src, dst string) error {
	if err := c.Copy(b, src, dst); err != nil {
		return err
	}

	if err := c.DeleteFile(b, src); err != nil {
		return err
	}

	return nil
}
