package gko

import (
	"cloud.google.com/go/bigquery"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/option"
	"google.golang.org/appengine"
)

var (
	_ BigQueryFactory = (*BigQueryFactoryImpl)(nil)
	_ BigQuery        = (*BigQueryClient)(nil)
)

// BigQueryFactory is bigquery factory interface.
type BigQueryFactory interface {
	New(context.Context) (BigQuery, error)
}

// BigQueryFactoryImpl is implementation of bigquery factory.
type BigQueryFactoryImpl struct{}

// New return bigquery client.
func (b *BigQueryFactoryImpl) New(ctx context.Context) (BigQuery, error) {
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

// BigQueryClient is bigquery client.
type BigQueryClient struct {
	ctx    context.Context
	client *bigquery.Client
}

// newBigQueryClient return new bigquery client.
func newBigQueryClient(ctx context.Context) (*BigQueryClient, error) {
	client, err := bigquery.NewClient(ctx, appengine.AppID(ctx), option.WithTokenSource(google.AppEngineTokenSource(ctx)))
	if err != nil {
		ErrorLog(ctx, err.Error())
		return nil, err
	}

	return &BigQueryClient{ctx, client}, nil
}

// Query run bigquery query, then return job.
func (b *BigQueryClient) Query(q string) (*bigquery.Job, error) {
	query := b.client.Query(q)
	query.UseStandardSQL = true
	return query.Run(b.ctx)
}

// CreateTable create table in dataset bigquery client have.
//
// This method always create table with standard sql option.
func (b *BigQueryClient) CreateTable(dataset, table string) error {
	return b.client.Dataset(dataset).Table(table).Create(b.ctx, bigquery.UseStandardSQL())
}

// DeleteTable delete table in dataset bigquery client have.
func (b *BigQueryClient) DeleteTable(dataset, table string) error {
	return b.client.Dataset(dataset).Table(table).Delete(b.ctx)
}

// UploadRow upload one or more row.
func (b *BigQueryClient) UploadRow(dataset, table, suffix string, src interface{}) error {
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
