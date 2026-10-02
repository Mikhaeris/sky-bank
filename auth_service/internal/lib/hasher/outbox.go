package hasher

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"uuid"

	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
)

const (
	outboxKeySize       = 32
	outboxCipherVersion = byte(1)
)

type EventCipher struct {
	aead cipher.AEAD
}

func NewEventCipher(pathToKey string) (*EventCipher, error) {
	key, err := os.ReadFile(pathToKey)
	if err != nil {
		return nil, fmt.Errorf("read outbox encryption key: %w", err)
	}
	defer clear(key)
	if len(key) != outboxKeySize {
		return nil, fmt.Errorf("invalid outbox encryption key length: %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create outbox cipher: %w", err)
	}

	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("create outbox AEAD: %w", err)
	}

	return &EventCipher{aead: aead}, nil
}

func (c *EventCipher) Encrypt(event ke.Event[ke.OtpPayload]) ([]byte, error) {
	if c == nil || c.aead == nil {
		return nil, errors.New("outbox cipher is not initialized")
	}
	if event.Metadata.EventID == uuid.Nil() || event.Metadata.AggregateID == uuid.Nil() {
		return nil, errors.New("outbox event and challenge IDs must be set")
	}

	plaintext, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox event: %w", err)
	}
	defer clear(plaintext)

	encrypted := []byte{outboxCipherVersion}
	encrypted = c.aead.Seal(encrypted, nil, plaintext, outboxAdditionalData(event.Metadata.EventID, event.Metadata.AggregateID))
	return encrypted, nil
}

func (c *EventCipher) Decrypt(encrypted []byte, eventID, challengeID uuid.UUID) (ke.Event[ke.OtpPayload], error) {
	var event ke.Event[ke.OtpPayload]
	if c == nil || c.aead == nil {
		return event, errors.New("outbox cipher is not initialized")
	}
	if eventID == uuid.Nil() || challengeID == uuid.Nil() {
		return event, errors.New("outbox event and challenge IDs must be set")
	}
	if len(encrypted) < 1+c.aead.Overhead() || encrypted[0] != outboxCipherVersion {
		return event, errors.New("invalid outbox ciphertext format")
	}

	plaintext, err := c.aead.Open(nil, nil, encrypted[1:], outboxAdditionalData(eventID, challengeID))
	if err != nil {
		return event, fmt.Errorf("decrypt outbox event: %w", err)
	}
	defer clear(plaintext)

	if err := json.Unmarshal(plaintext, &event); err != nil {
		return ke.Event[ke.OtpPayload]{}, fmt.Errorf("unmarshal outbox event: %w", err)
	}
	if event.Metadata.EventID != eventID || event.Metadata.AggregateID != challengeID {
		return ke.Event[ke.OtpPayload]{}, errors.New("outbox event IDs do not match ciphertext metadata")
	}

	return event, nil
}

func outboxAdditionalData(eventID, challengeID uuid.UUID) []byte {
	data := make([]byte, 0, len(eventID)+len(challengeID))
	data = append(data, eventID[:]...)
	return append(data, challengeID[:]...)
}
