package gko

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	pubsub "google.golang.org/api/pubsub/v1"
)

var _ CloudPubSub = (*CloudPubSubClient)(nil)

// Topic is cloud pubsub topic.
type Topic string

// PubSubMessage is cloud pubsub message.
type PubSubMessage struct {
	Data interface{}
	Attr map[string]string
}

// CloudPubSubFactory is cloud pub/sub factory interface.
type CloudPubSubFactory interface {
	New(context.Context) (CloudPubSub, error)
}

// CloudPubSubFactoryImpl implements cloud pub/sub factory interface.
type CloudPubSubFactoryImpl struct{}

// New return new cloud pub/sub client.
func (c *CloudPubSubFactoryImpl) New(ctx context.Context) (CloudPubSub, error) {
	return newCloudPubSubClient(ctx)
}

// CloudPubSub is cloud pub/sub interface.
type CloudPubSub interface {
	CreateTopic(Topic) error
	GetTopic(Topic) error
	DeleteTopic(Topic) error
	Publish(Topic, []*PubSubMessage) error
}

// CloudPubSubClient is cloud pubsub client
type CloudPubSubClient struct {
	ctx context.Context
	s   *pubsub.Service
}

// newCloudPubSubClient return new cloud pubsub client.
func newCloudPubSubClient(ctx context.Context) (*CloudPubSubClient, error) {
	client := oauth2.NewClient(ctx, google.AppEngineTokenSource(ctx, pubsub.PubsubScope))
	s, err := pubsub.New(client)
	if err != nil {
		ErrorLog(ctx, err.Error())
		return nil, err
	}

	return &CloudPubSubClient{ctx, s}, nil
}

// CreateTopic create cloud pubsub topic.
func (c *CloudPubSubClient) CreateTopic(topic Topic) error {
	t, err := c.s.Projects.Topics.Create(string(topic), &pubsub.Topic{}).Context(c.ctx).Do()
	if err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}

	if 400 <= t.HTTPStatusCode {
		ErrorLog(c.ctx, "status for creating topic is in error range")
		return fmt.Errorf("pubsub reponse error code")
	}

	if t.Name != string(topic) {
		ErrorLog(c.ctx, "created topic name is not equal to specified")
		return fmt.Errorf("pubsub topic name is wrong")
	}

	return nil
}

// GetTopic get cloud pubsub topic.
func (c *CloudPubSubClient) GetTopic(topic Topic) error {
	t, err := c.s.Projects.Topics.Get(string(topic)).Context(c.ctx).Do()
	if err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}

	if 400 <= t.HTTPStatusCode {
		ErrorLog(c.ctx, "status for creating topic is in error range")
		return fmt.Errorf("pubsub reponse error code")
	}

	if t.Name != string(topic) {
		ErrorLog(c.ctx, "created topic name is not equal to specified")
		return fmt.Errorf("pubsub topic name is wrong")
	}

	return nil
}

// DeleteTopic delete cloud pubsub topic.
func (c *CloudPubSubClient) DeleteTopic(topic Topic) error {
	if err := c.GetTopic(topic); err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}

	e, err := c.s.Projects.Topics.Delete(string(topic)).Context(c.ctx).Do()
	if err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}

	if 400 <= e.HTTPStatusCode {
		ErrorLog(c.ctx, "status for creating topic is in error range")
		return fmt.Errorf("pubsub reponse error code")
	}

	return nil
}

// Publish publish messages to cloud pubsub topic.
func (c *CloudPubSubClient) Publish(topic Topic, messages []*PubSubMessage) error {
	pm := make([]*pubsub.PubsubMessage, 0, len(messages))
	for _, m := range messages {
		j, err := json.Marshal(m.Data)
		if err != nil {
			ErrorLog(c.ctx, err.Error())
			return err
		}
		data := base64.StdEncoding.EncodeToString(j)
		pm = append(pm, &pubsub.PubsubMessage{Data: data, Attributes: m.Attr})
	}

	res, err := c.s.Projects.Topics.Publish(string(topic), &pubsub.PublishRequest{Messages: pm}).Context(c.ctx).Do()
	if err != nil {
		ErrorLog(c.ctx, err.Error())
		return err
	}

	if 400 <= res.HTTPStatusCode {
		ErrorLog(c.ctx, "status for creating topic is in error range")
		return fmt.Errorf("pubsub reponse error code")
	}

	return nil
}
