package gko

import (
	"errors"

	"fmt"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	_ CloudLoggingFactory = (*cloudLoggingFactoryImpl)(nil)
	_ CloudLogging        = (*cloudLoggingClient)(nil)
)

var cloudLoggingFactory CloudLoggingFactory

// CloudLoggingFactory is cloud logging factory interface.
type CloudLoggingFactory interface {
	New(context.Context, string, oauth2.TokenSource) (CloudLogging, error)
}

// cloudLoggingFactoryImpl implements cloud logging factory.
type cloudLoggingFactoryImpl struct{}

// CloudLogging is cloud logging interface.
type CloudLogging interface {
	Entries(filters []string, newestFirst bool, maxSize int, pageToken string) ([]*logging.Entry, string, bool, error)
	CreateSink(sinkID, dst, filter string) (*logadmin.Sink, error)
	DeleteSink(sinkID string) error
}

// cloudLoggingClient is cloud logging client.
type cloudLoggingClient struct {
	ctx    context.Context
	client *logging.Client
	admin  *logadmin.Client
}

// GetCloudLogginFactory return cloud logging factory.
func GetCloudLogginFactory() CloudLoggingFactory {
	if cloudLoggingFactory == nil {
		cloudLoggingFactory = &cloudLoggingFactoryImpl{}
	}
	return cloudLoggingFactory
}

// New return new cloud logging client.
//
// If ts is specified, replace default google token to specified token source.
func (c *cloudLoggingFactoryImpl) New(ctx context.Context, projectID string, ts oauth2.TokenSource) (CloudLogging, error) {
	var opts []option.ClientOption
	if ts != nil {
		opts = append(opts, option.WithTokenSource(ts))
	}
	client, err := logging.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	admin, err := logadmin.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	return &cloudLoggingClient{ctx, client, admin}, nil
}

// Entries return entry iterator.
//
// Options is loggin payload filters, updated date order and max size.
func (c *cloudLoggingClient) Entries(filters []string, newestFirst bool, maxSize int, pageToken string) ([]*logging.Entry, string, bool, error) {
	var opts []logadmin.EntriesOption
	for _, f := range filters {
		opts = append(opts, logadmin.Filter(f))
	}
	if newestFirst {
		opts = append(opts, logadmin.NewestFirst())
	}

	itr := c.admin.Entries(c.ctx, opts...)
	if maxSize > 0 {
		itr.PageInfo().MaxSize = maxSize
	}

	var entries []*logging.Entry
	pageToken, err := iterator.NewPager(itr, maxSize, pageToken).NextPage(&entries)
	if err != nil {
		return nil, "", false, err
	}

	return entries, pageToken, pageToken != "", nil
}

// CreateSink creates new cloud logging sink.
func (c *cloudLoggingClient) CreateSink(sinkID, dst, filter string) (*logadmin.Sink, error) {
	s, err := c.admin.Sink(c.ctx, sinkID)
	if err == nil {
		return nil, fmt.Errorf("%s is already exist", sinkID)
	}
	if err != nil {
		if gapierr, ok := err.(*googleapi.Error); ok {
			if gapierr.Code != 404 {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if dst == "" {
		return nil, errors.New("destination must not be empty")
	}

	s.Destination = dst
	s.Filter = filter

	return c.admin.CreateSink(c.ctx, s)
}

// DeleteSink deletes cloud logging sink.
func (c *cloudLoggingClient) DeleteSink(sinkID string) error {
	_, err := c.admin.Sink(c.ctx, sinkID)
	if err != nil {
		return err
	}

	return c.admin.DeleteSink(c.ctx, sinkID)
}
