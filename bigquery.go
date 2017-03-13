package gko

import (
	"cloud.google.com/go/bigquery"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/option"
	"google.golang.org/appengine"
)

var (
	_ BigQueryFactory = (*bigQueryFactoryImpl)(nil)
	_ BigQuery        = (*bigQueryClient)(nil)
)

var bigqueryFactory BigQueryFactory

// GetBigQueryFactory return bigquery factory.
func GetBigQueryFactory() BigQueryFactory {
	if bigqueryFactory == nil {
		bigqueryFactory = &bigQueryFactoryImpl{}
	}
	return bigqueryFactory
}

// BigQueryFactory is bigquery factory interface.
type BigQueryFactory interface {
	New(context.Context) (BigQuery, error)
}

// bigQueryFactoryImpl is implementation of bigquery factory.
type bigQueryFactoryImpl struct{}

// New return bigquery client.
func (b *bigQueryFactoryImpl) New(ctx context.Context) (BigQuery, error) {
	return newBigQueryClient(ctx)
}

// BigQuery is bigquery interface along with reader and writer.
type BigQuery interface {
	BigQueryReader
	BigQueryWriter
}

// BigQueryReader is bigquery reader interface.
type BigQueryReader interface {
	Query(string) (*bigquery.Job, error)
}

// BigQueryWriter is bigquery writer interface.
type BigQueryWriter interface {
	CreateTable(dataset, table string) error
	DeleteTable(dataset, table string) error
	UploadRow(dataset, table, suffix string, src interface{}) error
}

// bigQueryClient is bigquery client.
type bigQueryClient struct {
	ctx    context.Context
	client *bigquery.Client
}

// newBigQueryClient return new bigquery client.
func newBigQueryClient(ctx context.Context) (*bigQueryClient, error) {
	client, err := bigquery.NewClient(ctx, appengine.AppID(ctx), option.WithTokenSource(google.AppEngineTokenSource(ctx)))
	if err != nil {
		ErrorLog(ctx, err.Error())
		return nil, err
	}

	return &bigQueryClient{ctx, client}, nil
}

// Query run bigquery query, then return job.
func (b *bigQueryClient) Query(q string) (*bigquery.Job, error) {
	query := b.client.Query(q)
	query.UseStandardSQL = true
	return query.Run(b.ctx)
}

// CreateTable create table in dataset bigquery client have.
//
// This method always create table with standard sql option.
func (b *bigQueryClient) CreateTable(dataset, table string) error {
	return b.client.Dataset(dataset).Table(table).Create(b.ctx, bigquery.UseStandardSQL())
}

// DeleteTable delete table in dataset bigquery client have.
func (b *bigQueryClient) DeleteTable(dataset, table string) error {
	return b.client.Dataset(dataset).Table(table).Delete(b.ctx)
}

// UploadRow upload one or more row.
func (b *bigQueryClient) UploadRow(dataset, table, suffix string, src interface{}) error {
	t := b.client.Dataset(dataset).Table(table)
	// check src is valid
	if _, err := bigquery.InferSchema(src); err != nil {
		ErrorLog(b.ctx, err.Error())
		return err
	}

	upl := t.Uploader()
	upl.TableTemplateSuffix = suffix
	return upl.Put(b.ctx, src)
}
