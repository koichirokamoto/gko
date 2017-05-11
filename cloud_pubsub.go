package gko

import (
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

var (
	_ CloudPubSubFactory = (*cloudPubSubFactoryImpl)(nil)
	_ CloudPubSub        = (*cloudPubSubClient)(nil)
)

var cloudPubSubFactory CloudPubSubFactory

var errResourceNotFound = errors.New("resource is not found")

// Topic is cloud pubsub topic.
type Topic string

// SubscriptionConfig is cloud pubsub subscription config.
type SubscriptionConfig struct {
	PushConfig *pubsub.PushConfig
	Deadline   time.Duration
}

// CloudPubSubFactory is cloud pub/sub factory interface.
type CloudPubSubFactory interface {
	New(context.Context, string, oauth2.TokenSource) (CloudPubSub, error)
}

// cloudPubSubFactoryImpl implements cloud pub/sub factory interface.
type cloudPubSubFactoryImpl struct{}

// CloudPubSub is cloud pub/sub interface.
type CloudPubSub interface {
	CreateTopic(Topic) error
	DeleteTopic(Topic) error
	Publish(Topic, *pubsub.Message) error
	PushSubscription(Topic, time.Duration, *pubsub.PushConfig) error
	PullSubscription(Topic, time.Duration, func(context.Context, *pubsub.Message)) error
}

// cloudPubSubClient is cloud pubsub client
type cloudPubSubClient struct {
	ctx    context.Context
	client *pubsub.Client
}

// GetCloudPubSubFactory return cloud pub/sub factory.
func GetCloudPubSubFactory() CloudPubSubFactory {
	if cloudPubSubFactory == nil {
		cloudPubSubFactory = &cloudPubSubFactoryImpl{}
	}
	return cloudPubSubFactory
}

// New return new cloud pub/sub client.
//
// If ts is specified, replace default google token to specified token source.
func (c *cloudPubSubFactoryImpl) New(ctx context.Context, projectID string, ts oauth2.TokenSource) (CloudPubSub, error) {
	var opts []option.ClientOption
	if ts != nil {
		opts = append(opts, option.WithTokenSource(ts))
	}
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	return &cloudPubSubClient{ctx, client}, nil
}

// CreateTopic create cloud pubsub topic.
func (c *cloudPubSubClient) CreateTopic(topic Topic) error {
	t := c.client.Topic(string(topic))
	err := c.existsTopic(t)
	if err == nil || (err != nil && err != errResourceNotFound) {
		return err
	}

	_, err = c.client.CreateTopic(c.ctx, string(topic))
	if err != nil {
		return err
	}

	return nil
}

// existsTopic check cloud pubsub topic is created.
func (c *cloudPubSubClient) existsTopic(topic *pubsub.Topic) error {
	exists, err := topic.Exists(c.ctx)
	if err != nil {
		return err
	}

	if !exists {
		return errResourceNotFound
	}

	return nil
}

// DeleteTopic delete cloud pubsub topic.
func (c *cloudPubSubClient) DeleteTopic(topic Topic) error {
	t := c.client.Topic(string(topic))
	if err := c.existsTopic(t); err != nil {
		return err
	}

	if err := c.client.Topic(string(topic)).Delete(c.ctx); err != nil {
		return err
	}

	return nil
}

// Publish publish messages to cloud pubsub topic.
func (c *cloudPubSubClient) Publish(topic Topic, msg *pubsub.Message) error {
	_, err := c.client.Topic(string(topic)).Publish(c.ctx, msg).Get(c.ctx)
	if err != nil {
		return err
	}
	return nil
}

// PushSubscription crate cloud pubsub push subscription.
func (c *cloudPubSubClient) PushSubscription(topic Topic, deadline time.Duration, pushConfig *pubsub.PushConfig) error {
	if pushConfig == nil {
		return fmt.Errorf("subscription push config must not be null")
	}

	t := c.client.Topic(string(topic))
	err := c.existsTopic(t)
	if err != nil {
		return err
	}

	id := RandSeq(32)
	sub := c.client.Subscription(id)
	err = c.existsSubscription(sub)
	if err == nil || (err != nil && err != errResourceNotFound) {
		return err
	}

	_, err = c.client.CreateSubscription(c.ctx, id, t, deadline, pushConfig)
	if err != nil {
		return err
	}

	return nil
}

func (c *cloudPubSubClient) PullSubscription(topic Topic, deadline time.Duration, f func(context.Context, *pubsub.Message)) error {
	t := c.client.Topic(string(topic))
	err := c.existsTopic(t)
	if err != nil {
		return err
	}

	id := RandSeq(32)
	sub := c.client.Subscription(id)
	err = c.existsSubscription(sub)
	if err == nil || (err != nil && err != errResourceNotFound) {
		return err
	}

	sub, err = c.client.CreateSubscription(c.ctx, id, t, deadline, nil)
	if err != nil {
		return err
	}

	err = sub.Receive(c.ctx, f)
	if err != nil {
		return err
	}

	return nil
}

func (c *cloudPubSubClient) existsSubscription(sub *pubsub.Subscription) error {
	exists, err := sub.Exists(c.ctx)
	if err != nil {
		return err
	}

	if !exists {
		return errResourceNotFound
	}

	return nil
}
