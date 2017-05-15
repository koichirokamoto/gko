package gko

import (
	"errors"
	"fmt"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

// Entries return entry iterator.
//
// Options is loggin payload filters, updated date order and max size.
func Entries(ctx context.Context, admin *logadmin.Client, filters []string, newestFirst bool, maxSize int, pageToken string) ([]*logging.Entry, string, bool, error) {
	var opts []logadmin.EntriesOption
	for _, f := range filters {
		opts = append(opts, logadmin.Filter(f))
	}
	if newestFirst {
		opts = append(opts, logadmin.NewestFirst())
	}

	itr := admin.Entries(ctx, opts...)
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
func CreateSink(ctx context.Context, admin *logadmin.Client, sinkID, dst, filter string) (*logadmin.Sink, error) {
	s, err := admin.Sink(ctx, sinkID)
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

	return admin.CreateSink(ctx, s)
}

// DeleteSink deletes cloud logging sink.
func DeleteSink(ctx context.Context, admin *logadmin.Client, sinkID string) error {
	_, err := admin.Sink(ctx, sinkID)
	if err != nil {
		return err
	}

	return admin.DeleteSink(ctx, sinkID)
}
