package gko

import (
	"golang.org/x/net/context"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
)

var (
	_ CloudDatastoreFactory = (*cloudDatastoreFactoryImpl)(nil)
	_ CloudDatastore        = (*cloudDatastoreClient)(nil)
)

var cloudDatastoreFactory CloudDatastoreFactory

// GetCloudDatastoreFactory return cloudDatastore factory.
func GetCloudDatastoreFactory() CloudDatastoreFactory {
	if cloudDatastoreFactory == nil {
		cloudDatastoreFactory = &cloudDatastoreFactoryImpl{}
	}
	_, ok := cloudDatastoreFactory.(*cloudDatastoreFactoryImpl)
	if !ok {
		cloudDatastoreFactory = &cloudDatastoreFactoryImpl{}
	}
	return cloudDatastoreFactory
}

// CloudDatastoreFactory is cloudDatastore factory interface.
type CloudDatastoreFactory interface {
	New(context.Context) (CloudDatastore, error)
}

// gaeCloudDatastoreFactoryImpl is implementation of cloudDatastore factory.
type cloudDatastoreFactoryImpl struct{}

// New return cloudDatastore client.
func (b *cloudDatastoreFactoryImpl) New(ctx context.Context) (CloudDatastore, error) {
	return newCloudDatastoreClient(ctx)
}

// CloudDatastore is cloudDatastore interface along with reader and writer.
type CloudDatastore interface {
	GetAll(*datastore.Query, interface{}) ([]*datastore.Key, error)
}

// cloudDatastoreClient is cloud datastore client
type cloudDatastoreClient struct {
	ctx    context.Context
	client *datastore.Client
}

func newCloudDatastoreClient(ctx context.Context) (*cloudDatastoreClient, error) {
	t, projectID, err := getDefaultTokenSource(ctx, datastore.ScopeDatastore)
	if err != nil {
		return nil, err
	}

	client, err := datastore.NewClient(ctx, projectID, option.WithTokenSource(t))
	if err != nil {
		return nil, err
	}

	return &cloudDatastoreClient{ctx, client}, nil
}

// GetAll get all entities from cloud datastore.
func (c *cloudDatastoreClient) GetAll(q *datastore.Query, dst interface{}) ([]*datastore.Key, error) {
	return c.client.GetAll(c.ctx, q, dst)
}
