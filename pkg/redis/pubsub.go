package redis

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	goRedis "github.com/redis/go-redis/v9"
	"sync"
)

const buffchan = 100

type Subscriber interface {
	Channel() <-chan *Message
	Close() error
}

type subscriber struct {
	userChannel chan *Message   // НАШ канал, НЕ ЗАКРЫВАЕТСЯ ПРИ ПЕРЕПОДКЛЮЧЕНИИ
	pubsub      *goRedis.PubSub // текущий pubsub
	channels    []string        // подписанные каналы
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	closed      bool // флаг ручной проверки закрытия канала в действительности
}

// наш тип сообщения, но структура не полная (без pattern и тд)
type Message struct {
	Channel string
	Payload string
}

func (s *subscriber) Channel() <-chan *Message {
	return s.userChannel
}

func (s *subscriber) Close() error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.cancel()

	err := s.pubsub.Close()

	logger.Info(context.Background(), "redis subscriber closed",
		logger.Any("channels", s.channels),
	)

	return err
}

func (c *client) Subscribe(ctx context.Context, channels ...string) (Subscriber, error) {

	ps := c.universal().Subscribe(ctx, channels...)

	// ждем подтверждения подписки
	_, err := ps.Receive(ctx)
	if err != nil {
		logger.Error(ctx, "redis subscribe failed",
			logger.Any("channels", channels),
			logger.String("error", err.Error()),
		)
		return nil, err
	}

	subCtx, cancel := context.WithCancel(ctx)

	sub := &subscriber{
		userChannel: make(chan *Message, buffchan), // буферизованный канал TODO: проверить
		pubsub:      ps,
		channels:    channels,
		ctx:         subCtx,
		cancel:      cancel,
	}

	//регистрируем подписчика в клиенте
	c.registerSubscriber(sub)

	//запускаем пересылку сообщений
	go sub.forwardMessages()

	c.touchActivity()

	logger.Info(ctx, "redis subscribe success",
		logger.Any("channels", channels),
	)

	return sub, nil
}

func (c *client) registerSubscriber(sub *subscriber) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	c.subs[sub] = struct{}{}
}

func (c *client) unregisterSubscriber(sub *subscriber) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	delete(c.subs, sub)
}

func (s *subscriber) forwardMessages() {

	for {
		select {
		case msg, ok := <-s.pubsub.Channel():

			if !ok {
				if s.isClosed() {
					return // выходим при нормальном закрытии канала
				}
				logger.Warn(context.Background(), "redis pubsub channel closed unexpectedly",
					logger.Any("channels", s.channels),
				)
			}

			// пересылаем в наш канал
			select {
			case s.userChannel <- &Message{Channel: msg.Channel, Payload: msg.Payload}:

			case <-s.ctx.Done():
				return
			}

		case <-s.ctx.Done():
			return
		}
	}

}

func (s *subscriber) resubscribe(ctx context.Context, newClient goRedis.UniversalClient) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	//сохраняем старые информацию
	oldCancel := s.cancel
	oldPubsub := s.pubsub

	// создаем новый pubsub на НОВОМ клиенте
	newPubsub := newClient.Subscribe(ctx, s.channels...)
	if _, err := newPubsub.Receive(ctx); err != nil {
		_ = newPubsub.Close()
		logger.Error(ctx, "redis resubscribe failed",
			logger.Any("channels", s.channels),
			logger.String("error", err.Error()),
		)
		return fmt.Errorf("subscribe to %v: %w", s.channels, err)
	}

	//создаем новый контекст для новой горутины
	newCtx, newCancel := context.WithCancel(context.Background())

	//заменяем все
	s.pubsub = newPubsub
	s.ctx = newCtx
	s.cancel = newCancel

	//запускаем НОВУЮ горутину пересылки
	go s.forwardMessages()

	//останавливаем СТАРУЮ горутину
	go func() {
		//отменяем старый контекст - остановит старую forwardMessages()
		oldCancel()
		// Закрываем старый pubsub
		_ = oldPubsub.Close()
	}()

	logger.Info(ctx, "redis resubscribe success",
		logger.Any("channels", s.channels),
	)

	return nil
}

func (s *subscriber) updatePubsub(newPubsub *goRedis.PubSub) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		_ = newPubsub.Close() // подписчик уже закрыт
		logger.Info(context.Background(), "redis updatePubsub ignored (subscriber closed)",
			logger.Any("channels", s.channels),
		)
		return
	}

	// Заменяем pubsub
	oldPubsub := s.pubsub
	s.pubsub = newPubsub

	// Закрываем старый в отдельной горутине
	go func() {
		_ = oldPubsub.Close()
	}()

	// Запускаем новую горутину пересылки
	go s.forwardMessages()
	logger.Debug(context.Background(), "redis pubsub updated",
		logger.Any("channels", s.channels),
	)
}

// проверяем с мьютексом закрытие канала
func (s *subscriber) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
