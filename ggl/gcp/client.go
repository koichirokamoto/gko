package gcp

import (
	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

// NewDatastoreClient returns function to create new cloud datastore client.
func NewDatastoreClient(opts ...option.ClientOption) func(context.Context, string) (*datastore.Client, error) {
	return func(ctx context.Context, projectID string) (*datastore.Client, error) {
		return datastore.NewClient(ctx, projectID, opts...)
	}
}

// NewStorageClient returns function to create new cloud storage client.
func NewStorageClient(opts ...option.ClientOption) func(context.Context) (*storage.Client, error) {
	return func(ctx context.Context) (*storage.Client, error) {
		return storage.NewClient(ctx, opts...)
	}
}

// NewBigQueryClient returns function to create new bigquery client.
func NewBigQueryClient(opts ...option.ClientOption) func(context.Context, string) (*bigquery.Client, error) {
	return func(ctx context.Context, projectID string) (*bigquery.Client, error) {
		return bigquery.NewClient(ctx, projectID, opts...)
	}
}

// NewPubSubClient returns function to create new cloud pubsub client.
func NewPubSubClient(opts ...option.ClientOption) func(context.Context, string) (*pubsub.Client, error) {
	return func(ctx context.Context, projectID string) (*pubsub.Client, error) {
		return pubsub.NewClient(ctx, projectID, opts...)
	}
}

// NewLoggingClient returns function to create new stackdriver logging client.
func NewLoggingClient(opts ...option.ClientOption) func(context.Context, string) (*logging.Client, error) {
	return func(ctx context.Context, projectID string) (*logging.Client, error) {
		return logging.NewClient(ctx, projectID, opts...)
	}
}

// NewLoggingAdminClient returns function to create new stackdriver logging admin client.
func NewLoggingAdminClient(opts ...option.ClientOption) func(context.Context, string) (*logadmin.Client, error) {
	return func(ctx context.Context, projectID string) (*logadmin.Client, error) {
		return logadmin.NewClient(ctx, projectID, opts...)
	}
}
