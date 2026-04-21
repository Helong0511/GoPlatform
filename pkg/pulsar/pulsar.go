package pulsar

import (
	"context"
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"
)

// PulsarProducer Pulsar生产者
type PulsarProducer struct {
	client   pulsar.Client
	producer pulsar.Producer
}

// PulsarConsumer Pulsar消费者
type PulsarConsumer struct {
	Client   pulsar.Client
	Consumer pulsar.Consumer
}

// NewPulsarProducer 创建Pulsar生产者
func NewPulsarProducer(url, topic string) (*PulsarProducer, error) {
	// 创建Pulsar客户端
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pulsar client: %w", err)
	}

	// 创建生产者
	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: topic,
	})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create pulsar producer: %w", err)
	}

	return &PulsarProducer{
		client:   client,
		producer: producer,
	}, nil
}

// NewPulsarConsumer 创建Pulsar消费者
func NewPulsarConsumer(url, topic, subscription string) (*PulsarConsumer, error) {
	// 创建Pulsar客户端
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pulsar client: %w", err)
	}

	// 创建消费者
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subscription,
		Type:             pulsar.Shared,
	})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create pulsar consumer: %w", err)
	}

	return &PulsarConsumer{
		Client:   client,
		Consumer: consumer,
	}, nil
}

// Produce 发送消息
func (p *PulsarProducer) Produce(message []byte) error {
	_, err := p.producer.Send(context.Background(), &pulsar.ProducerMessage{
		Payload: message,
	})
	return err
}

// Consume 消费消息
func (p *PulsarConsumer) Consume() (pulsar.Message, error) {
	return p.Consumer.Receive(context.Background())
}

// Close 关闭生产者
func (p *PulsarProducer) Close() {
	p.producer.Close()
	p.client.Close()
}

// Close 关闭消费者
func (p *PulsarConsumer) Close() {
	p.Consumer.Close()
	p.Client.Close()
}
