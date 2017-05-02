package gko

import (
	"errors"
	"time"

	"fmt"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	_ CloudLoggingFactory = (*cloudLoggingFactoryImpl)(nil)
	_ CloudLogging        = (*cloudLoggingClient)(nil)
)

var cloudLoggingFactory CloudLoggingFactory

// GetCloudLogginFactory return cloud logging factory.
func GetCloudLogginFactory() CloudLoggingFactory {
	if cloudLoggingFactory == nil {
		cloudLoggingFactory = &cloudLoggingFactoryImpl{}
	}
	return cloudLoggingFactory
}

// CloudLoggingFactory is cloud logging factory interface.
type CloudLoggingFactory interface {
	New(context.Context) (CloudLogging, error)
}

// cloudLoggingFactoryImpl implements cloud logging factory.
type cloudLoggingFactoryImpl struct{}

// New return new cloud logging client.
func (c *cloudLoggingFactoryImpl) New(ctx context.Context) (CloudLogging, error) {
	return newCloudLogginClient(ctx)
}

// CloudLogging is cloud logging interface.
type CloudLogging interface {
	Send(logID, severity string, opts []logging.LoggerOption, payload interface{})
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

// newCloudLogginClient return new cloud logging client.
//
// This is assumed that user has admin scope of cloud logging.
func newCloudLogginClient(ctx context.Context) (*cloudLoggingClient, error) {
	t, projectID, err := getDefaultTokenSource(ctx, logging.AdminScope)
	if err != nil {
		return nil, err
	}

	ts := option.WithTokenSource(t)

	client, err := logging.NewClient(ctx, projectID, ts)
	if err != nil {
		return nil, err
	}

	admin, err := logadmin.NewClient(ctx, projectID, ts)
	if err != nil {
		return nil, err
	}

	return &cloudLoggingClient{ctx, client, admin}, nil
}

// Send send payload to cloud logging.
func (c *cloudLoggingClient) Send(logID, severity string, opts []logging.LoggerOption, payload interface{}) {
	l := c.client.Logger(logID, opts...)
	e := logging.Entry{
		InsertID:  RandSeq(32),
		Severity:  logging.ParseSeverity(severity),
		Timestamp: time.Now(),
		Payload:   payload,
	}
	l.Log(e)
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
