package gko

import (
	"time"

	"cloud.google.com/go/logging"
	"golang.org/x/net/context"
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
}

// cloudLoggingClient is cloud logging client.
type cloudLoggingClient struct {
	ctx    context.Context
	client *logging.Client
}

// newCloudLogginClient return new cloud logging client.
func newCloudLogginClient(ctx context.Context) (*cloudLoggingClient, error) {
	t, projectID, err := getDefaultTokenSource(ctx, logging.AdminScope)
	if err != nil {
		return nil, err
	}

	client, err := logging.NewClient(ctx, projectID, option.WithTokenSource(t))
	if err != nil {
		return nil, err
	}

	return &cloudLoggingClient{ctx, client}, nil
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
