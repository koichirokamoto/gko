package gcp

import (
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"

	"golang.org/x/net/context"
)

// CreateTopic create cloud pubsub topic.
func CreateTopic(ctx context.Context, client *pubsub.Client, topic string) error {
	t := client.Topic(topic)
	exists, err := t.Exists(ctx)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("topic %s is already exist", topic)
	}

	_, err = client.CreateTopic(ctx, topic)
	if err != nil {
		return err
	}

	return nil
}

// DeleteTopic delete cloud pubsub topic.
func DeleteTopic(ctx context.Context, client *pubsub.Client, topic string) error {
	t := client.Topic(topic)
	exists, err := t.Exists(ctx)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("topic %s is not already exist", topic)
	}

	if err := client.Topic(topic).Delete(ctx); err != nil {
		return err
	}

	return nil
}

// PushSubscription crate cloud pubsub push subscription.
func PushSubscription(ctx context.Context, client *pubsub.Client, topic string, deadline time.Duration, pushConfig *pubsub.PushConfig) error {
	if pushConfig == nil {
		return fmt.Errorf("subscription push config must not be null")
	}

	t := client.Topic(topic)
	exists, err := t.Exists(ctx)
	if err != nil {
		return err
	}

	if !exists {
		err = CreateTopic(ctx, client, topic)
		if err != nil {
			return err
		}
	}

	id := RandSeq(32)
	sub := client.Subscription(id)
	exists, err = sub.Exists(ctx)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("subscription %s is already exist", id)
	}

	_, err = client.CreateSubscription(ctx, id, t, deadline, pushConfig)
	if err != nil {
		return err
	}

	return nil
}

// PullSubscription pulls subscription from topic.
func PullSubscription(ctx context.Context, client *pubsub.Client, topic string, deadline time.Duration, f func(context.Context, *pubsub.Message)) error {
	t := client.Topic(topic)
	exists, err := t.Exists(ctx)
	if err != nil {
		return err
	}

	if !exists {
		err = CreateTopic(ctx, client, topic)
		if err != nil {
			return err
		}
	}

	id := RandSeq(32)
	sub := client.Subscription(id)
	exists, err = sub.Exists(ctx)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("subscription %s is already exist", id)
	}

	sub, err = client.CreateSubscription(ctx, id, t, deadline, nil)
	if err != nil {
		return err
	}

	err = sub.Receive(ctx, f)
	if err != nil {
		return err
	}

	return nil
}
