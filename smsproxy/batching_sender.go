package smsproxy

import (
	"sync"
	"time"

	"gitlab.com/devskiller-tasks/messaging-app-golang/fastsmsing"
)

type batchingClient interface {
	send(message SendMessage, ID MessageID) error
}

func newBatchingClient(
	repository repository,
	client fastsmsing.FastSmsingClient,
	config smsProxyConfig,
	statistics ClientStatistics,
) batchingClient {
	return &simpleBatchingClient{
		repository:     repository,
		client:         client,
		messagesToSend: make([]fastsmsing.Message, 0),
		config:         config,
		statistics:     statistics,
		lock:           sync.RWMutex{},
	}
}

type simpleBatchingClient struct {
	config         smsProxyConfig
	repository     repository
	client         fastsmsing.FastSmsingClient
	statistics     ClientStatistics
	messagesToSend []fastsmsing.Message
	lock           sync.RWMutex
}

func (b *simpleBatchingClient) send(message SendMessage, ID MessageID) error {
	if err := b.repository.save(ID); err != nil {
		return err
	}

	b.lock.Lock()
	b.messagesToSend = append(b.messagesToSend, fastsmsing.Message{
		MessageID:   ID,
		Message:     message.Message,
		PhoneNumber: message.PhoneNumber,
	})

	if len(b.messagesToSend) >= b.config.minimumInBatch {
		routeineMessages := make([]fastsmsing.Message, len(b.messagesToSend))
		copy(routeineMessages, b.messagesToSend)
		b.messagesToSend = make([]fastsmsing.Message, 0)
		b.lock.Unlock()

		go func(messagesToSend []fastsmsing.Message) {
			for attempt := 1; attempt <= calculateMaxAttempts(b.config.maxAttempts); attempt++ {
				err := b.client.Send(messagesToSend)
				time.Sleep(3 * time.Second)
				if lastAttemptFailed(attempt, b.config.maxAttempts, err) {
					go sendStatistics(messagesToSend, err, attempt, b.config.maxAttempts, b.statistics)
					return
				}
			}
			go sendStatistics(messagesToSend, nil, b.config.maxAttempts, b.config.maxAttempts, b.statistics)
		}(routeineMessages)
	} else {
		b.lock.Unlock()
	}

	return nil
}

func calculateMaxAttempts(configMaxAttempts int) int {
	if configMaxAttempts < 1 {
		return 1
	}
	return configMaxAttempts
}

func lastAttemptFailed(currentAttempt int, maxAttempts int, currentAttemptError error) bool {
	result := currentAttempt == maxAttempts && currentAttemptError != nil
	return result
}

func sendStatistics(messages []fastsmsing.Message, lastErr error, currentAttempt int, maxAttempts int, statistics ClientStatistics) {
	statistics.Send(clientResult{
		messagesBatch:  messages,
		err:            lastErr,
		currentAttempt: currentAttempt,
		maxAttempts:    maxAttempts,
	})
}
