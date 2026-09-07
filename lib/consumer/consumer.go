package consumer

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	retryInitial = 500 * time.Millisecond
	retryMax     = 30 * time.Second
)

type Consumer struct {
	reader    *kafka.Reader
	projector *Projector
}

func New(brokers []string, topics []string, groupID string, projector *Projector) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			GroupTopics: topics,
			GroupID:     groupID,
			// Offsets are committed explicitly, after the write lands.
			CommitInterval: 0,
		}),
		projector: projector,
	}
}

func (c *Consumer) Close() error { return c.reader.Close() }

func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		// TODO: run async
		if err := c.applyWithRetry(ctx, msg); err != nil {
			return err
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

func (c *Consumer) applyWithRetry(ctx context.Context, msg kafka.Message) error {
	backoff := retryInitial
	for {
		err := c.projector.Apply(ctx, msg.Value)
		if err == nil {
			return nil
		}

		var unprocessable ErrUnprocessable
		if errors.As(err, &unprocessable) {
			log.Printf("consumer: skipping unprocessable message at offset %d: %v",
				msg.Offset, err)
			return nil
		}

		log.Printf("consumer: retrying offset %d in %s: %v", msg.Offset, backoff, err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > retryMax {
			backoff = retryMax
		}
	}
}
