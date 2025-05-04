package tronclient

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

const (
	tronGridAPI = "https://api.shasta.trongrid.io"
)

type TronHTTPClient struct {
	APIKey       string
	USDTContract string
}

func NewTronHTTPClient(apiKey string, usdtContract string) *TronHTTPClient {
	return &TronHTTPClient{
		APIKey:       apiKey,
		USDTContract: usdtContract,
	}
}

func (c *TronHTTPClient) SendUSDT(fromPrivKey string, toAddress string, amount float64) (string, error) {
	fromAddr, fromAddrHex, privKey, err := getTronAddressAndHexFromPrivKey(fromPrivKey)
	if err != nil {
		return "", err
	}

	fmt.Println("=== SendUSDT DEBUG ===")
	fmt.Println("From Address:", fromAddr)
	fmt.Println("To Address:", toAddress)
	fmt.Println("Contract Address (base58):", c.USDTContract)
	fmt.Println("Amount:", amount)

	contractHex := base58CheckToHex(c.USDTContract)
	params := encodeTransferParams(toAddress, amount)

	fmt.Println("From (hex):", fromAddrHex)
	fmt.Println("To (hex):", base58CheckToHex(toAddress))
	fmt.Println("Contract (hex):", contractHex)
	fmt.Println("Encoded Params:", params)
	fmt.Println("======================")

	param := map[string]interface{}{
		"owner_address":     fromAddrHex,
		"contract_address":  contractHex,
		"function_selector": "transfer(address,uint256)",
		"parameter":         params,
		"call_value":        0,
		"fee_limit":         2_000_000,
		"visible":           false,
	}

	rawTx, err := c.post("/wallet/triggersmartcontract", param)
	if err != nil {
		return "", err
	}

	fmt.Println("RAW TX (triggersmartcontract):", string(rawTx))

	signedTx, err := signTransaction(rawTx, privKey)
	if err != nil {
		return "", err
	}

	broadcastResult, err := c.post("/wallet/broadcasttransaction", signedTx)
	if err != nil {
		return "", err
	}

	fmt.Println("Broadcast result:", string(broadcastResult))

	if !strings.Contains(string(broadcastResult), "txid") {
		return "", errors.New("broadcast failed: " + string(broadcastResult))
	}

	var res map[string]interface{}
	_ = json.Unmarshal(broadcastResult, &res)
	return res["txid"].(string), nil
}

func (c *TronHTTPClient) GetUSDTBalance(address string) (float64, error) {
	decoded, err := base58.Decode(address)
	if err != nil || len(decoded) != 25 {
		return 0, fmt.Errorf("invalid TRON address: %v", err)
	}

	addr := decoded[:21] // 21 байт без чексума
	addrBody := addr[1:] // без префикса 0x41
	addrHex := hex.EncodeToString(addr)

	param := map[string]interface{}{
		"owner_address":     addrHex,
		"contract_address":  base58CheckToHex(c.USDTContract),
		"function_selector": "balanceOf(address)",
		"parameter":         fmt.Sprintf("%064x", addrBody),
		"visible":           false,
	}

	response, err := c.post("/wallet/triggerconstantcontract", param)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return 0, err
	}

	constants, ok := result["constant_result"].([]interface{})
	if !ok || len(constants) == 0 {
		return 0, errors.New("empty constant_result")
	}

	hexStr, _ := constants[0].(string)
	balance := new(big.Int)
	balance.SetString(hexStr, 16)

	return float64(balance.Int64()) / 1e6, nil
}
func (c *TronHTTPClient) post(path string, payload interface{}) ([]byte, error) {
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", tronGridAPI+path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TRON-PRO-API-KEY", c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func base58CheckToHex(address string) string {
	decoded, err := base58.Decode(address)
	if err != nil {
		panic(fmt.Sprintf("INVALID base58 address: %s — %v", address, err))
	}

	// Убираем последние 4 байта чексума
	if len(decoded) != 25 {
		panic(fmt.Sprintf("INVALID base58check address length: %s — got %d bytes", address, len(decoded)))
	}

	raw := decoded[:21] // ← TRON address = prefix(1) + body(20)
	return hex.EncodeToString(raw)
}

func encodeTransferParams(toAddress string, amount float64) string {
	decoded, err := base58.Decode(toAddress)
	if err != nil {
		panic(fmt.Sprintf("invalid toAddress base58: %s", err))
	}

	// Убираем чексума (25 → 21 байт), затем префикс 0x41
	if len(decoded) != 25 {
		panic("invalid address length — must be 25 (base58check)")
	}
	raw := decoded[:21]
	if raw[0] != 0x41 {
		panic("invalid TRON address prefix")
	}
	addrBody := raw[1:] // 20 байт

	amt := big.NewInt(int64(amount * 1e6))
	return fmt.Sprintf("%064x%064x", addrBody, amt)
}

func getTronAddressAndHexFromPrivKey(privHex string) (string, string, *ecdsa.PrivateKey, error) {
	privBytes, err := hex.DecodeString(privHex)
	if err != nil {
		return "", "", nil, err
	}

	privKey, err := crypto.ToECDSA(privBytes)
	if err != nil {
		return "", "", nil, err
	}

	pubKey := privKey.PublicKey
	pubBytes := crypto.FromECDSAPub(&pubKey)[1:]
	hash := crypto.Keccak256(pubBytes)
	addr := append([]byte{0x41}, hash[12:]...)

	first := sha256.Sum256(addr)
	second := sha256.Sum256(first[:])
	checksum := second[:4]
	full := append(addr, checksum...)

	return base58.Encode(full), hex.EncodeToString(addr), privKey, nil
}

func signTransaction(rawTx []byte, privKey *ecdsa.PrivateKey) (map[string]interface{}, error) {
	var tx map[string]interface{}
	if err := json.Unmarshal(rawTx, &tx); err != nil {
		return nil, err
	}

	// Проверка наличия "transaction"
	txMap, ok := tx["transaction"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing transaction field in rawTx response")
	}

	rawData, err := json.Marshal(txMap["raw_data"])
	if err != nil {
		return nil, err
	}

	hash := crypto.Keccak256(rawData)
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		return nil, err
	}

	txMap["signature"] = []string{hex.EncodeToString(sig)}
	return txMap, nil
}
