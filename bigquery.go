package gko

import (
	"errors"

	"cloud.google.com/go/bigquery"

	"golang.org/x/net/context"

	"fmt"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	_ BigQueryFactory = (*bigQueryFactoryImpl)(nil)
	_ BigQuery        = (*bigQueryClient)(nil)
)

var (
	errTableAlreadyExist   = errors.New("table is already exist")
	errDatasetAlreadyExist = errors.New("dataset is already exist")
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
	CreateTable(dataset, table string, useStdSQL bool) error
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

// CreateDataset creates new bigquery dataset.
//
// If dataset is already exist, return error.
func (b *bigQueryClient) CreateDataset(dataset string) error {
	if err := b.checkDatasetExist(dataset); err != nil {
		return err
	}
	return b.client.Dataset(dataset).Create(b.ctx)
}

func (b *bigQueryClient) DeleteDataset(dataset string) error {
	if err := b.checkDatasetExist(dataset); err != errDatasetAlreadyExist {
		if err == nil {
			return fmt.Errorf("%s is not already exist", dataset)
		}
		return err
	}
	return b.client.Dataset(dataset).Delete(b.ctx)
}

// CreateTable create table in dataset bigquery client have.
//
// This method always create table with standard sql option.
// If table is already exist in dataset, return error.
func (b *bigQueryClient) CreateTable(dataset, table string, useStdSQL bool) error {
	if err := b.checkTableExist(dataset, table); err != nil {
		return err
	}
	var opts []bigquery.CreateTableOption
	if useStdSQL {
		opts = append(opts, bigquery.UseStandardSQL())
	}
	return b.client.Dataset(dataset).Table(table).Create(b.ctx, opts...)
}

// DeleteTable delete table in dataset bigquery client have.
func (b *bigQueryClient) DeleteTable(dataset, table string) error {
	if err := b.checkTableExist(dataset, table); err != errTableAlreadyExist {
		if err == nil {
			return fmt.Errorf("%s:%s is not already exist", dataset, table)
		}
		return err
	}
	return b.client.Dataset(dataset).Table(table).Delete(b.ctx)
}

// UploadRow upload one or more row.
func (b *bigQueryClient) UploadRow(dataset, table, suffix string, src interface{}) error {
	t := b.client.Dataset(dataset).Table(table)
	upl := t.Uploader()
	upl.TableTemplateSuffix = suffix
	return upl.Put(b.ctx, src)
}

// checkTableExist checks bigquery table is already exist.
//
// If table is already exist, return error.
func (b *bigQueryClient) checkTableExist(dataset, table string) error {
	_, err := b.client.Dataset(dataset).Table(table).Metadata(b.ctx)
	if err != nil {
		// if status code is not 404, return error because of exist
		if gapierr, ok := err.(*googleapi.Error); ok && gapierr.Code == 404 {
			return nil
		}
		return err
	}
	return errTableAlreadyExist
}

// checkDatasetExist checks bigquery dataset is already exist.
//
// If dataset is already exist, return error.
func (b *bigQueryClient) checkDatasetExist(dataset string) error {
	_, err := b.client.Dataset(dataset).Metadata(b.ctx)
	if err != nil {
		// if status code is not 404, return error because of exist
		if gapierr, ok := err.(*googleapi.Error); ok && gapierr.Code == 404 {
			return nil
		}
		return err
	}
	return errDatasetAlreadyExist
}
