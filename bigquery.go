package gko

import (
	"cloud.google.com/go/bigquery"

	"golang.org/x/net/context"

	"google.golang.org/api/option"
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
	_, ok := bigqueryFactory.(*bigQueryFactoryImpl)
	if !ok {
		bigqueryFactory = &bigQueryFactoryImpl{}
	}
	return bigqueryFactory
}

// BigQueryFactory is bigquery factory interface.
type BigQueryFactory interface {
	New(context.Context) (BigQuery, error)
}

// gaeBigQueryFactoryImpl is implementation of bigquery factory.
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
	Query(string, bool) (*bigquery.Job, error)
	GetQueryResult(*bigquery.Job) (*bigquery.RowIterator, error)
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
	t, projectID, err := getDefaultTokenSource(ctx, bigquery.Scope)
	if err != nil {
		return nil, err
	}

	client, err := bigquery.NewClient(ctx, projectID, option.WithTokenSource(t))
	if err != nil {
		return nil, err
	}

	return &bigQueryClient{ctx, client}, nil
}

// Query run bigquery query, then return job.
func (b *bigQueryClient) Query(q string, useStdSQL bool) (*bigquery.Job, error) {
	query := b.client.Query(q)
	query.UseStandardSQL = useStdSQL
	return query.Run(b.ctx)
}

func (b *bigQueryClient) GetQueryResult(job *bigquery.Job) (*bigquery.RowIterator, error) {
	stat, err := job.Wait(b.ctx)
	if err != nil {
		return nil, err
	}

	if stat.Err() != nil {
		return nil, stat.Err()
	}

	return job.Read(b.ctx)
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
		return err
	}

	upl := t.Uploader()
	upl.TableTemplateSuffix = suffix
	return upl.Put(b.ctx, src)
}
