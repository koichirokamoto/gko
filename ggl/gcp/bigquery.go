package gcp

import (
	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
)

// GetQueryResult gets query result of job.
func GetQueryResult(ctx context.Context, job *bigquery.Job) (*bigquery.RowIterator, error) {
	stat, err := job.Wait(ctx)
	if err != nil {
		return nil, err
	}

	if stat.Err() != nil {
		return nil, stat.Err()
	}

	return job.Read(ctx)
}
