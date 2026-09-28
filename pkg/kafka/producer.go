package kafka

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var ErrUnknownType = errors.New("unknown type")

type Producer struct {
	producer     *kafka.Producer
	flushTimeout int
}

func NewProducer(address []string) (*Producer, error) {
	cfg := &kafka.ConfigMap{
		"bootstrap.servers":        strings.Join(address, ","),
		"allow.auto.create.topics": false,
	}
	p, err := kafka.NewProducer(cfg)
	if err != nil {
		return nil, err
	}
	return &Producer{producer: p}, nil
}

func (p *Producer) Produce(ctx context.Context, topic string, evt kafkaevents.BaseEvent) error {
	eventData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	metadata := evt.GetMetadata()

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   metadata.AggregateID[:],
		Value: eventData,
	}

	ctx, span := otel.Tracer("sky-bank/pkg/kafka").Start(
		ctx,
		"kafka.send "+topic,
		trace.WithSpanKind(trace.SpanKindProducer),
	)
	defer span.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	for key, value := range carrier {
		message.Headers = append(message.Headers, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}

	ch := make(chan kafka.Event)
	err = p.producer.Produce(message, ch)
	if err != nil {
		return err
	}

	e := <-ch
	switch ev := e.(type) {
	case *kafka.Message:
		return ev.TopicPartition.Error
	case kafka.Error:
		return ev
	default:
		return ErrUnknownType
	}
}

func (p *Producer) Close() {
	p.producer.Flush(p.flushTimeout)
	p.producer.Close()
}
