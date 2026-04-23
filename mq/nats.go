// Package mq 提供 NATS JetStream 消息总线的统一封装，
// 供秒杀异步下单、库存扣减通知等场景使用。
package mq

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config 描述 NATS JetStream 连接与 Stream 配置
type Config struct {
	URL      string   // NATS 连接地址，如 "nats://localhost:4222"
	Stream   string   // Stream 名称，如 "SECKILL"
	Subjects []string // Stream 监听的 subject 模式，如 []string{"seckill.>"}
}

// Handler 是消息处理函数，返回 nil 表示处理成功（会 Ack），返回 error 会 Nak（触发重投）
type Handler func(ctx context.Context, payload []byte) error

// Bus 是对 NATS JetStream 的封装，提供 Publish / Subscribe / Close 三个操作
type Bus struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	str jetstream.Stream
}

// New 建立 NATS 连接并确保 Stream 存在（不存在则创建，已存在则更新配置）
func New(cfg Config) (*Bus, error) {
	nc, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	str, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     cfg.Stream,
		Subjects: cfg.Subjects,
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		return nil, fmt.Errorf("stream: %w", err)
	}
	return &Bus{nc: nc, js: js, str: str}, nil
}

// Publish 向指定 subject 发布一条消息
func (b *Bus) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := b.js.Publish(ctx, subject, data)
	return err
}

// Subscribe 创建一个持久消费者并开始消费。
// consumer 是持久化名称（重启后继续从上次位置消费），subject 是过滤条件。
func (b *Bus) Subscribe(ctx context.Context, consumer, subject string, h Handler) error {
	c, err := b.str.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       consumer,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}
	_, err = c.Consume(func(msg jetstream.Msg) {
		if err := h(ctx, msg.Data()); err != nil {
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	return err
}

// Close 关闭 NATS 连接
func (b *Bus) Close() { b.nc.Close() }
