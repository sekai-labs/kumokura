package keyring

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	zk "github.com/zalando/go-keyring"
)

var (
	ErrNotFound = errors.New("secret not found")
)

type Store interface {
	Get(service, user string) (string, error)
	Set(service, user, secret string) error
	Delete(service, user string) error
}

type SecureKeyring struct {
	fallback Store
}

func NewSecureKeyring(secretsDir string) *SecureKeyring {
	return &SecureKeyring{
		fallback: NewFileEncryptedStore(secretsDir),
	}
}

func (k *SecureKeyring) Get(service, user string) (string, error) {
	val, err := zk.Get(service, user)
	if err == nil {
		return val, nil
	}
	return k.fallback.Get(service, user)
}

func (k *SecureKeyring) Set(service, user, secret string) error {
	err := zk.Set(service, user, secret)
	if err == nil {
		_ = k.fallback.Set(service, user, secret)
		return nil
	}
	return k.fallback.Set(service, user, secret)
}

func (k *SecureKeyring) Delete(service, user string) error {
	zErr := zk.Delete(service, user)
	fErr := k.fallback.Delete(service, user)
	if zErr == nil || fErr == nil {
		return nil
	}
	if errors.Is(fErr, ErrNotFound) {
		return ErrNotFound
	}
	return fErr
}

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]string),
	}
}

func (m *MemoryStore) Get(service, user string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", service, user)
	val, ok := m.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return val, nil
}

func (m *MemoryStore) Set(service, user, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", service, user)
	m.data[key] = secret
	return nil
}

func (m *MemoryStore) Delete(service, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", service, user)
	if _, ok := m.data[key]; !ok {
		return ErrNotFound
	}
	delete(m.data, key)
	return nil
}

type FileEncryptedStore struct {
	mu         sync.RWMutex
	secretsDir string
	filePath   string
	key        []byte
}

func NewFileEncryptedStore(secretsDir string) *FileEncryptedStore {
	if secretsDir == "" {
		home, _ := os.UserHomeDir()
		secretsDir = filepath.Join(home, ".config", "kumokura", "secrets")
	}
	salt := []byte("kumokura-v1-storage-local-salt")
	hash := sha256.Sum256(append([]byte(secretsDir), salt...))
	return &FileEncryptedStore{
		secretsDir: secretsDir,
		filePath:   filepath.Join(secretsDir, "vault.enc"),
		key:        hash[:],
	}
}

func (f *FileEncryptedStore) load() (map[string]string, error) {
	if err := os.MkdirAll(f.secretsDir, 0700); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(f.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return make(map[string]string), nil
	}

	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var data map[string]string
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (f *FileEncryptedStore) save(data map[string]string) error {
	if err := os.MkdirAll(f.secretsDir, 0700); err != nil {
		return err
	}
	plaintext, err := json.Marshal(data)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(f.key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(f.filePath, ciphertext, 0600)
}

func (f *FileEncryptedStore) Get(service, user string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	data, err := f.load()
	if err != nil {
		return "", err
	}
	key := hex.EncodeToString([]byte(fmt.Sprintf("%s:%s", service, user)))
	val, ok := data[key]
	if !ok {
		return "", ErrNotFound
	}
	return val, nil
}

func (f *FileEncryptedStore) Set(service, user, secret string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := f.load()
	if err != nil {
		return err
	}
	key := hex.EncodeToString([]byte(fmt.Sprintf("%s:%s", service, user)))
	data[key] = secret
	return f.save(data)
}

func (f *FileEncryptedStore) Delete(service, user string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := f.load()
	if err != nil {
		return err
	}
	key := hex.EncodeToString([]byte(fmt.Sprintf("%s:%s", service, user)))
	if _, ok := data[key]; !ok {
		return ErrNotFound
	}
	delete(data, key)
	return f.save(data)
}
