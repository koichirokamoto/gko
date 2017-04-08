package gko

import (
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"

	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	_ CloudPubSubFactory = (*cloudPubSubFactoryImpl)(nil)
	_ CloudPubSub        = (*cloudPubSubClient)(nil)
)

var cloudPubSubFactory CloudPubSubFactory

// GetCloudPubSubFactory return cloud pub/sub factory.
func GetCloudPubSubFactory() CloudPubSubFactory {
	if cloudPubSubFactory == nil {
		cloudPubSubFactory = &cloudPubSubFactoryImpl{}
	}
	return cloudPubSubFactory
}

// Topic is cloud pubsub topic.
type Topic string

// SubscriptionConfig is cloud pubsub subscription config.
type SubscriptionConfig struct {
	PushConfig *pubsub.PushConfig
	Deadline   time.Duration
}

// CloudPubSubFactory is cloud pub/sub factory interface.
type CloudPubSubFactory interface {
	New(context.Context) (CloudPubSub, error)
}

// cloudPubSubFactoryImpl implements cloud pub/sub factory interface.
type cloudPubSubFactoryImpl struct{}

// New return new cloud pub/sub client.
func (c *cloudPubSubFactoryImpl) New(ctx context.Context) (CloudPubSub, error) {
	return newCloudPubSubClient(ctx)
}

// CloudPubSub is cloud pub/sub interface.
type CloudPubSub interface {
	CreateTopic(Topic) error
	DeleteTopic(Topic) error
	Publish(Topic, *pubsub.Message) error
	PushSubscription(Topic, time.Duration, *pubsub.PushConfig) error
	PullSubscription(Topic, ...pubsub.PullOption) ([]*pubsub.Message, error)
}

// cloudPubSubClient is cloud pubsub client
type cloudPubSubClient struct {
	ctx    context.Context
	client *pubsub.Client
}

// newCloudPubSubClient return new cloud pubsub client.
func newCloudPubSubClient(ctx context.Context) (*cloudPubSubClient, error) {
	t, projectID, err := getDefaultTokenSource(ctx, pubsub.ScopePubSub)
	if err != nil {
		return nil, err
	}

	client, err := pubsub.NewClient(ctx, projectID, option.WithTokenSource(t))
	if err != nil {
		return nil, err
	}

	return &cloudPubSubClient{ctx, client}, nil
}

// CreateTopic create cloud pubsub topic.
func (c *cloudPubSubClient) CreateTopic(topic Topic) error {
	t, err := c.existsTopic(topic)
	if err != nil {
		return err
	}

	if t != nil {
		return fmt.Errorf("topic %s is already created", topic)
	}

	_, err = c.client.CreateTopic(c.ctx, string(topic))
	if err != nil {
		return err
	}

	return nil
}

// existsTopic check cloud pubsub topic is created.
func (c *cloudPubSubClient) existsTopic(topic Topic) (*pubsub.Topic, error) {
	t := c.client.Topic(string(topic))
	exists, err := t.Exists(c.ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("topic %s is not found", topic)
	}

	return t, nil
}

// DeleteTopic delete cloud pubsub topic.
func (c *cloudPubSubClient) DeleteTopic(topic Topic) error {
	if _, err := c.existsTopic(topic); err != nil {
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

	t, err := c.existsTopic(topic)
	if err != nil {
		return err
	}

	id := RandSeq(32)
	sub, err := c.existsSubscription(id)
	if sub != nil {
		return fmt.Errorf("subscription %s is already created", id)
	}

	_, err = c.client.CreateSubscription(c.ctx, id, t, deadline, pushConfig)
	if err != nil {
		return err
	}

	return nil
}

func (c *cloudPubSubClient) PullSubscription(topic Topic, opt ...pubsub.PullOption) ([]*pubsub.Message, error) {
	_, err := c.existsTopic(topic)
	if err != nil {
		return nil, err
	}

	id := RandSeq(32)
	sub, err := c.existsSubscription(id)
	if err != nil {
		return nil, err
	}

	if sub != nil {
		return nil, fmt.Errorf("subscription %s is already created", id)
	}

	itr, err := sub.Pull(c.ctx, opt...)
	if err != nil {
		return nil, err
	}

	var messages []*pubsub.Message

	for {
		msg, err := itr.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		msg.Done(true)
		messages = append(messages, msg)
	}

	return messages, nil
}

func (c *cloudPubSubClient) existsSubscription(id string) (*pubsub.Subscription, error) {
	sub := c.client.Subscription(id)
	exists, err := sub.Exists(c.ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("subscription %s is not found", id)
	}

	return sub, nil
}
