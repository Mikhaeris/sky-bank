package kafka

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Handler func(ctx context.Context, value []byte) error

func JSONHandler[T any](fn func(ctx context.Context, event T) error) Handler {
	return func(ctx context.Context, value []byte) error {
		var ev T
		if err := json.Unmarshal(value, &ev); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		return fn(ctx, ev)
	}
}

var sessionTimeout = 7000

type Consumer struct {
	consumer *kafka.Consumer
	logger   *slog.Logger
	handler  Handler
	stop     bool
}

func NewConsumer(
	logger *slog.Logger,
	brokers []string,
	consumerGroup string,
	topic string,
	h Handler,
) (*Consumer, error) {
	err := TestConnection(brokers, &topic)
	if err != nil {
		return nil, fmt.Errorf("kafka connection test failed: %w", err)
	}
	cfg := &kafka.ConfigMap{
		"bootstrap.servers":        strings.Join(brokers, ","),
		"group.id":                 consumerGroup,
		"session.timeout.ms":       sessionTimeout,
		"enable.auto.offset.store": false,
		"enable.auto.commit":       true,
		"auto.commit.interval.ms":  5000,
		"auto.offset.reset":        "earliest",
	}

	c, err := kafka.NewConsumer(cfg)
	if err != nil {
		return nil, err
	}

	err = c.Subscribe(topic, nil)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		consumer: c,
		logger:   logger,
		handler:  h,
		stop:     false,
	}, nil
}

func TestConnection(brokers []string, topic *string) error {
	if len(brokers) == 0 {
		return errors.New("no Kafka brokers configured")
	}
	cfg := &kafka.ConfigMap{
		"bootstrap.servers":        strings.Join(brokers, ","),
		"allow.auto.create.topics": false,
	}
	client, err := kafka.NewAdminClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Kafka client: %w", err)
	}
	defer client.Close()

	metadata, err := client.GetMetadata(topic, false, 5000)
	if err != nil {
		return fmt.Errorf("get Kafka metadata: %w", err)
	}
	if topic != nil {
		topicMetadata, ok := metadata.Topics[*topic]
		if !ok {
			return fmt.Errorf("Kafka metadata does not include topic %q", *topic)
		}
		if topicMetadata.Error.Code() != kafka.ErrNoError {
			return fmt.Errorf("Kafka topic %q: %w", *topic, topicMetadata.Error)
		}
	}

	return nil
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := c.consumer.ReadMessage(time.Second)
		if err != nil {
			var kerr kafka.Error
			if errors.As(err, &kerr) && kerr.Code() == kafka.ErrTimedOut {
				continue
			}
			c.logger.Error("read message", "error", err)
			continue
		}

		carrier := propagation.MapCarrier{}
		for _, header := range msg.Headers {
			carrier[header.Key] = string(header.Value)
		}

		msgCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
		processCtx, span := otel.Tracer("sky-bank/pkg/kafka").Start(
			msgCtx,
			"kafka.process",
			trace.WithSpanKind(trace.SpanKindConsumer),
		)
		err = c.handler(processCtx, msg.Value)
		span.End()
		if err != nil {
			c.logger.Error("handel message", "error", err)
			continue
		}

		_, err = c.consumer.StoreMessage(msg)
		if err != nil {
			c.logger.Error("can't store kafka msg", "error", err)
			continue
		}
	}
}

func (c *Consumer) Stop() error {
	return c.consumer.Close()
}
