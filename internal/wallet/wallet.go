package wallet

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

// Wallet структура с приватным ключом и адресом
type Wallet struct {
	PrivateKey string
	Address    string
}

// GenerateTRONWallet генерирует новый TRON-кошелек
func GenerateTRONWallet() (*Wallet, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	privBytes := crypto.FromECDSA(privateKey) // ← Ключ в правильной форме
	privHex := hex.EncodeToString(privBytes)

	address, err := generateTronAddress(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		PrivateKey: privHex,
		Address:    address,
	}, nil
}

// generateTronAddress получает TRON-адрес из публичного ключа
func generateTronAddress(pub *ecdsa.PublicKey) (string, error) {
	pubBytes := crypto.FromECDSAPub(pub)[1:]
	if len(pubBytes) != 64 {
		return "", errors.New("invalid public key length")
	}

	hash := crypto.Keccak256(pubBytes)
	addr := hash[12:]
	tronAddr := append([]byte{0x41}, addr...)

	// Чексума по стандарту TRON (double SHA256)
	first := sha256.Sum256(tronAddr)
	second := sha256.Sum256(first[:])
	checkSum := second[:4]

	// Адрес + чексума
	full := append(tronAddr, checkSum...)

	return base58.Encode(full), nil
}
