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
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

const (
	tronGridAPI = "https://api.trongrid.io"
	tronScanAPI = "https://tronscan.org"
)

// Alternative endpoints in case main one is down
var alternativeEndpoints = []string{
	"https://api.trongrid.io",
	"https://api.shasta.trongrid.io", // Testnet (for testing)
}

type TronHTTPClient struct {
	APIKey       string
	USDTContract string
}

func NewTronHTTPClient(apiKey string, usdtContract string) *TronHTTPClient {
	fmt.Printf("=== NewTronHTTPClient DEBUG ===\n")
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Printf("USDT Contract: %s\n", usdtContract)

	client := &TronHTTPClient{
		APIKey:       apiKey,
		USDTContract: usdtContract,
	}

	fmt.Printf("Client created with contract: %s\n", client.USDTContract)
	fmt.Printf("=== NewTronHTTPClient COMPLETED ===\n")

	return client
}

func (c *TronHTTPClient) SendUSDT(fromPrivKey string, toAddress string, amount float64) (string, error) {
	// Get the sender's address from private key
	fromAddr, fromAddrHex, privKey, err := getTronAddressAndHexFromPrivKey(fromPrivKey)
	if err != nil {
		return "", fmt.Errorf("failed to get address from private key: %v", err)
	}

	fmt.Println("=== SendUSDT DEBUG ===")
	fmt.Println("From Address:", fromAddr)
	fmt.Println("To Address:", toAddress)
	fmt.Println("Contract Address (base58):", c.USDTContract)
	fmt.Printf("Amount: %.6f\n", amount)

	// Check USDT balance first
	balance, err := c.GetUSDTBalance(fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to check USDT balance: %v", err)
	}
	fmt.Printf("Current USDT balance: %.6f\n", balance)

	if balance < amount {
		return "", fmt.Errorf("insufficient USDT balance: have %.6f, need %.6f", balance, amount)
	}

	// Estimate energy required
	estimatedEnergy, err := c.EstimateTransferEnergy(fromAddr, toAddress, amount)
	if err != nil {
		return "", fmt.Errorf("failed to estimate energy: %v", err)
	}
	fmt.Printf("Estimated energy required: %d\n", estimatedEnergy)

	// Check TRX balance
	trxBalance, err := c.GetTRXBalance(fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to check TRX balance: %v", err)
	}
	fmt.Printf("Current TRX balance: %.6f\n", trxBalance)

	// Calculate required TRX for energy
	// Using current TRON energy price: 420 SUN per energy
	const energyPrice = 420 // SUN per energy unit
	requiredTRX := float64(estimatedEnergy) * float64(energyPrice) / 1_000_000
	fmt.Printf("Required TRX for energy: %.6f\n", requiredTRX)

	// Add 20% buffer for safety (reduced from 50%)
	requiredTRXWithBuffer := requiredTRX * 1.2

	// If TRX balance is insufficient, try with minimal fee limit
	if trxBalance < requiredTRXWithBuffer {
		fmt.Printf("Warning: TRX balance (%.6f) is less than estimated requirement (%.6f)\n", trxBalance, requiredTRXWithBuffer)

		// Try with available TRX balance (minus small safety margin)
		availableTRX := trxBalance * 0.95 // Use 95% of available balance
		if availableTRX < 0.001 {         // Minimum 0.001 TRX
			return "", fmt.Errorf("insufficient TRX balance for energy: have %.6f TRX, need at least 0.001 TRX", trxBalance)
		}

		fmt.Printf("Using available TRX: %.6f (instead of estimated %.6f)\n", availableTRX, requiredTRXWithBuffer)
		requiredTRXWithBuffer = availableTRX
	}

	// Convert addresses to hex
	contractHex := base58CheckToHex(c.USDTContract)
	toAddrHex := base58CheckToHex(toAddress)

	fmt.Println("From (hex):", fromAddrHex)
	fmt.Println("To (hex):", toAddrHex)
	fmt.Println("Contract (hex):", contractHex)

	// Encode transfer parameters
	params := encodeTransferParams(toAddress, amount)
	fmt.Println("Encoded Params:", params)

	// Create transaction parameters with dynamic fee limit
	feeLimitSun := int64(requiredTRXWithBuffer * 1_000_000) // Convert TRX to SUN

	// Ensure minimum fee limit for USDT transfers
	if feeLimitSun < 2_000_000 { // If less than 2 TRX
		fmt.Printf("Setting minimum fee limit: 2 TRX (estimated: %.6f TRX)\n", requiredTRXWithBuffer)
		feeLimitSun = 2_000_000 // 2 TRX in SUN (минимальный лимит для USDT)
	} else if feeLimitSun > 5_000_000 { // If more than 5 TRX
		fmt.Printf("Warning: Fee limit too high (%.6f TRX), using maximum limit\n", requiredTRXWithBuffer)
		feeLimitSun = 5_000_000 // 5 TRX in SUN (максимальный лимит)
	}

	param := map[string]interface{}{
		"owner_address":     fromAddrHex,
		"contract_address":  contractHex,
		"function_selector": "transfer(address,uint256)",
		"parameter":         params[8:], // Remove methodID
		"call_value":        0,
		"fee_limit":         feeLimitSun,
		"visible":           false,
	}

	fmt.Println("=== Transaction Parameters ===")
	paramJSON, _ := json.MarshalIndent(param, "", "  ")
	fmt.Println(string(paramJSON))
	fmt.Println("===========================")

	// Create transaction with increased timeout
	client := &http.Client{
		Timeout: time.Second * 60, // Increase timeout to 60 seconds
	}

	var rawTx []byte
	maxRetries := 5 // Increase retries
	fmt.Printf("Creating transaction with %d retries...\n", maxRetries)
	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Creating transaction attempt %d/%d...\n", i+1, maxRetries)
		rawTx, err = c.postWithClient(client, "/wallet/triggersmartcontract", param)
		if err == nil {
			fmt.Printf("Transaction created successfully on attempt %d\n", i+1)
			break
		}
		fmt.Printf("Attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			fmt.Printf("Waiting 5 seconds before retry...\n")
			time.Sleep(time.Second * 5) // Increase delay between retries
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to create transaction after %d attempts: %v", maxRetries, err)
	}

	fmt.Println("RAW TX (triggersmartcontract):", string(rawTx))

	// Sign transaction
	var signedTx map[string]interface{}
	for i := 0; i < maxRetries; i++ {
		signedTx, err = signTransaction(rawTx, privKey)
		if err == nil {
			break
		}
		fmt.Printf("Signing attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * 1)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction after %d attempts: %v", maxRetries, err)
	}

	// Add delay before broadcasting
	time.Sleep(time.Second * 2)

	// Broadcast transaction with increased timeout
	var broadcastResult []byte
	broadcastClient := &http.Client{
		Timeout: time.Second * 120, // Увеличиваем timeout до 120 секунд
	}

	fmt.Printf("Broadcasting transaction with %d retries...\n", maxRetries)
	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Broadcast attempt %d/%d...\n", i+1, maxRetries)
		broadcastResult, err = c.postWithClient(broadcastClient, "/wallet/broadcasttransaction", signedTx)
		if err == nil {
			fmt.Printf("Broadcast successful on attempt %d\n", i+1)
			break
		}
		fmt.Printf("Broadcast attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			fmt.Printf("Waiting 10 seconds before retry...\n")
			time.Sleep(time.Second * 10) // Увеличиваем задержку между попытками
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction after %d attempts: %v", maxRetries, err)
	}

	fmt.Println("Broadcast result:", string(broadcastResult))

	// Parse broadcast result
	var result map[string]interface{}
	if err := json.Unmarshal(broadcastResult, &result); err != nil {
		return "", fmt.Errorf("failed to parse broadcast result: %v", err)
	}

	// Check for errors in broadcast result
	if code, ok := result["code"].(string); ok && code != "" {
		message := ""
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return "", fmt.Errorf("broadcast failed with code %s: %s", code, message)
	}

	// Get transaction ID
	txID, ok := result["txid"].(string)
	if !ok {
		return "", fmt.Errorf("invalid txid in response: %v", result)
	}

	return txID, nil
}

func (c *TronHTTPClient) GetUSDTBalance(address string) (float64, error) {
	fmt.Printf("=== GetUSDTBalance DEBUG ===\n")
	fmt.Printf("Address: %s\n", address)
	fmt.Printf("USDT Contract: %s\n", c.USDTContract)

	decoded, err := base58.Decode(address)
	if err != nil || len(decoded) != 25 {
		return 0, fmt.Errorf("invalid TRON address: %v", err)
	}

	addr := decoded[:21] // 21 байт без чексума
	addrBody := addr[1:] // без префикса 0x41
	addrHex := hex.EncodeToString(addr)

	fmt.Printf("Address (hex): %s\n", addrHex)
	fmt.Printf("Address body (hex): %x\n", addrBody)

	param := map[string]interface{}{
		"owner_address":     addrHex,
		"contract_address":  base58CheckToHex(c.USDTContract),
		"function_selector": "balanceOf(address)",
		"parameter":         fmt.Sprintf("%064x", addrBody),
		"visible":           false,
	}

	fmt.Printf("Request params: %+v\n", param)

	response, err := c.post("/wallet/triggerconstantcontract", param)
	if err != nil {
		fmt.Printf("API call failed: %v\n", err)
		return 0, err
	}

	fmt.Printf("API response: %s\n", string(response))

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		fmt.Printf("JSON unmarshal failed: %v\n", err)
		return 0, err
	}

	fmt.Printf("Parsed result: %+v\n", result)

	constants, ok := result["constant_result"].([]interface{})
	if !ok || len(constants) == 0 {
		fmt.Printf("Empty constant_result: %+v\n", result)
		return 0, errors.New("empty constant_result")
	}

	hexStr, _ := constants[0].(string)
	fmt.Printf("Balance hex: %s\n", hexStr)

	balance := new(big.Int)
	balance.SetString(hexStr, 16)

	usdtBalance := float64(balance.Int64()) / 1e6
	fmt.Printf("USDT Balance: %.6f\n", usdtBalance)
	fmt.Printf("=== GetUSDTBalance COMPLETED ===\n")

	return usdtBalance, nil
}

func (c *TronHTTPClient) post(path string, payload interface{}) ([]byte, error) {
	b, _ := json.Marshal(payload)
	url := tronGridAPI + path
	fmt.Printf("POST %s\nPayload: %s\n", url, string(b))

	client := &http.Client{
		Timeout: time.Second * 120, // увеличиваем таймаут до 2 минут
	}

	// Retry logic for TronGrid API calls
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Attempt %d/%d for %s\n", attempt, maxRetries, path)

		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("TRON-PRO-API-KEY", c.APIKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			fmt.Printf("HTTP error (attempt %d): %v\n", attempt, err)

			if attempt < maxRetries {
				fmt.Printf("Waiting 5 seconds before retry...\n")
				time.Sleep(time.Second * 5)
				continue
			}
			return nil, err
		}
		defer resp.Body.Close()

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			fmt.Printf("Read body error (attempt %d): %v\n", attempt, err)

			if attempt < maxRetries {
				fmt.Printf("Waiting 5 seconds before retry...\n")
				time.Sleep(time.Second * 5)
				continue
			}
			return nil, err
		}

		fmt.Printf("Response from %s (attempt %d): %s\n", path, attempt, string(responseBody))
		return responseBody, nil
	}

	return nil, fmt.Errorf("failed after %d attempts, last error: %v", maxRetries, lastErr)
}

func (c *TronHTTPClient) postWithClient(client *http.Client, path string, payload interface{}) ([]byte, error) {
	if client == nil {
		client = &http.Client{
			Timeout: time.Second * 180, // увеличиваем до 3 минут
		}
	}

	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", tronGridAPI+path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TRON-PRO-API-KEY", c.APIKey)

	resp, err := client.Do(req)
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
	// Method ID for transfer(address,uint256)
	methodID := "a9059cbb"

	decoded, err := base58.Decode(toAddress)
	if err != nil {
		panic(fmt.Sprintf("invalid toAddress base58: %s", err))
	}

	if len(decoded) != 25 {
		panic("invalid address length — must be 25 (base58check)")
	}

	raw := decoded[:21]
	if raw[0] != 0x41 {
		panic("invalid TRON address prefix")
	}

	// Remove the 0x41 prefix and pad to 32 bytes
	addrParam := fmt.Sprintf("%064x", raw[1:])

	// Convert amount to sun (6 decimals) and pad to 32 bytes
	amountInt := big.NewInt(int64(amount * 1e6))
	amountParam := fmt.Sprintf("%064x", amountInt)

	// Combine method ID and parameters
	return methodID + addrParam + amountParam
}

func getTronAddressAndHexFromPrivKey(privHex string) (string, string, *ecdsa.PrivateKey, error) {
	fmt.Println("=== Getting TRON Address from Private Key ===")
	fmt.Printf("Private Key (hex): %s\n", privHex)

	privBytes, err := hex.DecodeString(privHex)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to decode private key hex: %v", err)
	}

	privKey, err := crypto.ToECDSA(privBytes)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to convert to ECDSA: %v", err)
	}

	pubKey := privKey.PublicKey
	pubBytes := crypto.FromECDSAPub(&pubKey)
	fmt.Printf("Public Key (hex): %x\n", pubBytes)

	pubKeyHash := crypto.Keccak256(pubBytes[1:])
	fmt.Printf("Public Key Hash (hex): %x\n", pubKeyHash)

	addr := append([]byte{0x41}, pubKeyHash[12:]...)
	fmt.Printf("TRON Address (hex): %x\n", addr)

	// Calculate checksum
	first := sha256.Sum256(addr)
	second := sha256.Sum256(first[:])
	checksum := second[:4]
	fmt.Printf("Checksum (hex): %x\n", checksum)

	full := append(addr, checksum...)
	fmt.Printf("Full Address with Checksum (hex): %x\n", full)

	base58Addr := base58.Encode(full)
	fmt.Printf("Base58 Address: %s\n", base58Addr)
	fmt.Printf("Hex Address: %s\n", hex.EncodeToString(addr))
	fmt.Println("=== Address Generation Complete ===")

	return base58Addr, hex.EncodeToString(addr), privKey, nil
}

func signTransaction(rawTx []byte, privKey *ecdsa.PrivateKey) (map[string]interface{}, error) {
	var tx map[string]interface{}
	if err := json.Unmarshal(rawTx, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw transaction: %v", err)
	}

	// Для TRX transfer нет поля transaction, используем сам объект
	txObj := tx
	if t, ok := tx["transaction"].(map[string]interface{}); ok {
		txObj = t
	}

	fmt.Println("=== Signing Transaction ===")
	fmt.Printf("Raw Transaction: %+v\n", txObj)

	// Get raw_data_hex
	rawDataHex, ok := txObj["raw_data_hex"].(string)
	if !ok {
		return nil, errors.New("missing raw_data_hex in transaction")
	}

	fmt.Printf("Raw Data Hex to Sign: %s\n", rawDataHex)

	// Decode hex string to bytes
	rawDataBytes, err := hex.DecodeString(rawDataHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode raw_data_hex: %v", err)
	}

	// Create hash of the raw data
	hash := sha256.Sum256(rawDataBytes)
	fmt.Printf("Hash to Sign (hex): %x\n", hash)

	// Get the address from the private key for verification
	pubKey := privKey.Public().(*ecdsa.PublicKey)
	pubKeyBytes := crypto.FromECDSAPub(pubKey)
	address := crypto.Keccak256(pubKeyBytes[1:])[12:]

	// Convert to TRON address format
	tronAddress := append([]byte{0x41}, address...)
	fmt.Printf("Signing with TRON address (hex): %x\n", tronAddress)

	// Sign the hash
	sig, err := crypto.Sign(hash[:], privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	fmt.Printf("Signature (hex): %x\n", sig)

	// Add signature to transaction
	txObj["signature"] = []string{hex.EncodeToString(sig)}

	signedJSON, _ := json.MarshalIndent(txObj, "", "  ")
	fmt.Printf("Signed Transaction: %s\n", string(signedJSON))
	fmt.Println("=== Signing Complete ===")

	return txObj, nil
}

// ApproveUSDT approves the contract to spend USDT tokens
func (c *TronHTTPClient) ApproveUSDT(fromPrivKey string, spenderAddress string, amount float64) (string, error) {
	// Get the sender's address from private key
	fromAddr, fromAddrHex, privKey, err := getTronAddressAndHexFromPrivKey(fromPrivKey)
	if err != nil {
		return "", fmt.Errorf("failed to get address from private key: %v", err)
	}

	fmt.Println("=== ApproveUSDT DEBUG ===")
	fmt.Println("From Address:", fromAddr)
	fmt.Println("From Address Hex:", fromAddrHex)
	fmt.Println("Spender Address:", spenderAddress)
	fmt.Println("Contract Address (base58):", c.USDTContract)
	fmt.Printf("Amount: %.6f\n", amount)

	// Convert addresses to hex
	contractHex := base58CheckToHex(c.USDTContract)
	spenderHex := base58CheckToHex(spenderAddress)

	fmt.Println("Contract Hex:", contractHex)
	fmt.Println("Spender Hex:", spenderHex)

	// Method ID for approve(address,uint256)
	methodID := "095ea7b3"

	// Pad spender address to 32 bytes (remove 41 prefix and pad)
	spenderBytes, err := hex.DecodeString(spenderHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode spender hex: %v", err)
	}
	if len(spenderBytes) < 21 {
		return "", fmt.Errorf("invalid spender address length")
	}
	spenderParam := fmt.Sprintf("%064x", spenderBytes[1:]) // Remove 41 prefix and pad

	// Convert amount to sun (6 decimals) and pad to 32 bytes
	amountInt := big.NewInt(int64(amount * 1e6))
	amountParam := fmt.Sprintf("%064x", amountInt)

	// Combine parameters
	params := methodID + spenderParam + amountParam

	fmt.Println("Method ID:", methodID)
	fmt.Println("Spender Param:", spenderParam)
	fmt.Println("Amount Param:", amountParam)
	fmt.Println("Full Params:", params)

	// Create transaction parameters
	param := map[string]interface{}{
		"owner_address":     fromAddrHex,
		"contract_address":  contractHex,
		"function_selector": "approve(address,uint256)",
		"parameter":         params[8:], // Remove methodID as it's included in the data field
		"call_value":        0,
		"fee_limit":         2_000_000,
		"visible":           false,
	}

	paramJSON, _ := json.MarshalIndent(param, "", "  ")
	fmt.Println("Transaction Parameters:", string(paramJSON))

	// Create transaction with increased timeout
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	var rawTx []byte
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		rawTx, err = c.postWithClient(client, "/wallet/triggersmartcontract", param)
		if err == nil {
			break
		}
		fmt.Printf("Attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * 3)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to create transaction after %d attempts: %v", maxRetries, err)
	}

	fmt.Println("RAW TX (triggersmartcontract):", string(rawTx))

	// Parse raw transaction to verify its structure
	var rawTxMap map[string]interface{}
	if err := json.Unmarshal(rawTx, &rawTxMap); err != nil {
		return "", fmt.Errorf("failed to parse raw transaction: %v", err)
	}

	// Sign transaction
	var signedTx map[string]interface{}
	for i := 0; i < maxRetries; i++ {
		signedTx, err = signTransaction(rawTx, privKey)
		if err == nil {
			break
		}
		fmt.Printf("Signing attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * 2)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction after %d attempts: %v", maxRetries, err)
	}

	signedJSON, _ := json.MarshalIndent(signedTx, "", "  ")
	fmt.Println("Signed Transaction:", string(signedJSON))

	// Add delay before broadcasting
	time.Sleep(time.Second * 3)

	// Broadcast transaction
	var broadcastResult []byte
	for i := 0; i < maxRetries; i++ {
		broadcastResult, err = c.postWithClient(client, "/wallet/broadcasttransaction", signedTx)
		if err == nil {
			break
		}
		fmt.Printf("Broadcast attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * 4)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction after %d attempts: %v", maxRetries, err)
	}

	fmt.Println("Broadcast result:", string(broadcastResult))

	// Parse broadcast result
	var result map[string]interface{}
	if err := json.Unmarshal(broadcastResult, &result); err != nil {
		return "", fmt.Errorf("failed to parse broadcast result: %v", err)
	}

	// Check for errors in broadcast result
	if code, ok := result["code"].(string); ok && code != "" {
		message := ""
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return "", fmt.Errorf("broadcast failed with code %s: %s", code, message)
	}

	// Get transaction ID
	txID, ok := result["txid"].(string)
	if !ok {
		return "", fmt.Errorf("invalid txid in response: %v", result)
	}

	return txID, nil
}

// GetTransactionStatus проверяет статус транзакции
func (c *TronHTTPClient) GetTransactionStatus(txID string) (map[string]interface{}, error) {
	response, err := c.post("/wallet/gettransactionbyid", map[string]interface{}{
		"value":   txID,
		"visible": true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result, nil
}

// GetTRXBalance gets the TRX balance of an address
func (c *TronHTTPClient) GetTRXBalance(address string) (float64, error) {
	// Try TronGrid first, fallback to TronScan if it fails
	balance, err := c.getTRXBalanceFromTronGrid(address)
	if err != nil {
		fmt.Printf("TronGrid failed: %v, trying TronScan...\n", err)
		balance, err = c.getTRXBalanceFromTronScan(address)
		if err != nil {
			fmt.Printf("TronScan also failed: %v, returning 0 balance\n", err)
			// Return 0 balance if both APIs fail (account might not exist)
			return 0, nil
		}
	}
	return balance, nil
}

func (c *TronHTTPClient) getTRXBalanceFromTronScan(address string) (float64, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/account/balance/%s", tronScanAPI, address), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance from TronScan: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("TronScan balance response: %s\n", string(body))

	var result struct {
		Balance float64 `json:"balance"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse TronScan response: %v", err)
	}

	return result.Balance / 1_000_000, nil
}

func (c *TronHTTPClient) getTRXBalanceFromTronGrid(address string) (float64, error) {
	// Convert address to hex
	addrHex := base58CheckToHex(address)

	// Create request parameters
	param := map[string]interface{}{
		"address": addrHex,
		"visible": false,
	}

	// Call API to get account info
	response, err := c.post("/wallet/getaccount", param)
	if err != nil {
		return 0, fmt.Errorf("failed to get account info: %v", err)
	}

	fmt.Printf("TronGrid balance response: %s\n", string(response))

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v", err)
	}

	// Try different possible balance fields
	var balance float64

	// Try balance as string first
	if balStr, ok := result["balance"].(string); ok {
		balInt64, err := strconv.ParseInt(balStr, 10, 64)
		if err == nil {
			balance = float64(balInt64)
		}
	}

	// Try balance as float64
	if balance == 0 {
		if balFloat, ok := result["balance"].(float64); ok {
			balance = balFloat
		}
	}

	// Try balance as json.Number
	if balance == 0 {
		if balNum, ok := result["balance"].(json.Number); ok {
			if balFloat, err := balNum.Float64(); err == nil {
				balance = balFloat
			}
		}
	}

	// If account doesn't exist or has no balance, the API might return empty response
	if len(result) == 0 || balance == 0 {
		fmt.Printf("Warning: Account %s might not exist or has 0 balance\n", address)
		return 0, nil
	}

	// Convert from SUN to TRX
	return balance / 1_000_000, nil
}

// EstimateTransferEnergy estimates the energy required for a USDT transfer
func (c *TronHTTPClient) EstimateTransferEnergy(fromAddr string, toAddr string, amount float64) (int64, error) {
	// Convert addresses to hex
	fromAddrHex := base58CheckToHex(fromAddr)
	contractHex := base58CheckToHex(c.USDTContract)

	// Encode transfer parameters
	params := encodeTransferParams(toAddr, amount)

	// Create transaction parameters for energy estimation
	param := map[string]interface{}{
		"owner_address":     fromAddrHex,
		"contract_address":  contractHex,
		"function_selector": "transfer(address,uint256)",
		"parameter":         params[8:], // Remove methodID
		"visible":           false,      // Changed to false to use hex format
	}

	// Call API to estimate energy
	response, err := c.post("/wallet/triggerconstantcontract", param)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate energy: %v", err)
	}

	fmt.Printf("Energy estimation response: %s\n", string(response))

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v", err)
	}

	// The energy information might be nested in the result
	energyUsed, ok := result["energy_used"].(float64)
	if !ok {
		// Try to get it from transaction.ret[0].energy_usage
		if transaction, ok := result["transaction"].(map[string]interface{}); ok {
			if ret, ok := transaction["ret"].([]interface{}); ok && len(ret) > 0 {
				if retFirst, ok := ret[0].(map[string]interface{}); ok {
					if energy, ok := retFirst["energy_usage"].(float64); ok {
						energyUsed = energy
					}
				}
			}
		}
	}

	if energyUsed == 0 {
		// Use a more realistic default value for USDT transfers
		// Typical USDT transfer uses around 15,000-25,000 energy
		energyUsed = 20000
		fmt.Println("Warning: Using default energy estimation of 20,000")
	}

	// Add 20% buffer for safety
	return int64(energyUsed * 1.2), nil
}

// EstimateRequiredTRX estimates the TRX required for a USDT transfer
func (c *TronHTTPClient) EstimateRequiredTRX(fromAddr string, toAddr string, amount float64) (float64, error) {
	// Estimate energy required
	estimatedEnergy, err := c.EstimateTransferEnergy(fromAddr, toAddr, amount)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate energy: %v", err)
	}

	// Calculate required TRX for energy
	// Using current TRON energy price: 420 SUN per energy
	const energyPrice = 420 // SUN per energy unit
	requiredTRX := float64(estimatedEnergy) * float64(energyPrice) / 1_000_000

	// Use the actual estimated TRX amount
	fmt.Printf("Estimated TRX required: %.6f (based on %d energy units)\n", requiredTRX, estimatedEnergy)

	// Add 20% buffer for safety
	requiredTRXWithBuffer := requiredTRX * 1.2

	return requiredTRXWithBuffer, nil
}

// SendTRXForGas sends a small amount of TRX to cover gas fees
func (c *TronHTTPClient) SendTRXForGas(fromPrivKey string, toAddress string, amount float64) (string, error) {
	// Get the sender's address from private key
	fromAddr, fromAddrHex, privKey, err := getTronAddressAndHexFromPrivKey(fromPrivKey)
	if err != nil {
		return "", fmt.Errorf("failed to get address from private key: %v", err)
	}

	fmt.Println("=== SendTRXForGas DEBUG ===")
	fmt.Println("From Address:", fromAddr)
	fmt.Println("To Address:", toAddress)
	fmt.Printf("Amount: %.6f TRX\n", amount)

	// Check TRX balance first
	balance, err := c.GetTRXBalance(fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to check TRX balance: %v", err)
	}
	fmt.Printf("Current TRX balance: %.6f\n", balance)

	if balance < amount {
		return "", fmt.Errorf("insufficient TRX balance: have %.6f, need %.6f", balance, amount)
	}

	// Convert amount to SUN
	amountSun := int64(amount * 1_000_000)

	// Create transaction parameters
	param := map[string]interface{}{
		"owner_address": fromAddrHex,
		"to_address":    base58CheckToHex(toAddress),
		"amount":        amountSun,
		"visible":       false,
	}

	fmt.Println("=== TRX Transfer Parameters ===")
	paramJSON, _ := json.MarshalIndent(param, "", "  ")
	fmt.Println(string(paramJSON))
	fmt.Println("===========================")

	// Create transaction
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	var rawTx []byte
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		rawTx, err = c.postWithClient(client, "/wallet/createtransaction", param)
		if err == nil {
			break
		}
		fmt.Printf("Attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * 2)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to create TRX transaction: %v", err)
	}

	fmt.Println("RAW TX (createtransaction):", string(rawTx))

	// Sign transaction
	var signedTx map[string]interface{}
	for i := 0; i < maxRetries; i++ {
		signedTx, err = signTransaction(rawTx, privKey)
		if err == nil {
			break
		}
		fmt.Printf("Signing attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * 1)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to sign TRX transaction: %v", err)
	}

	// Broadcast transaction
	var broadcastResult []byte
	for i := 0; i < maxRetries; i++ {
		broadcastResult, err = c.post("/wallet/broadcasttransaction", signedTx)
		if err == nil {
			break
		}
		fmt.Printf("Broadcast attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Second * 2)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to broadcast TRX transaction: %v", err)
	}

	fmt.Println("Broadcast result:", string(broadcastResult))

	// Parse broadcast result
	var result map[string]interface{}
	if err := json.Unmarshal(broadcastResult, &result); err != nil {
		return "", fmt.Errorf("failed to parse broadcast result: %v", err)
	}

	// Check for errors in broadcast result
	if code, ok := result["code"].(string); ok && code != "" {
		message := ""
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return "", fmt.Errorf("broadcast failed with code %s: %s", code, message)
	}

	// Get transaction ID
	txID, ok := result["txid"].(string)
	if !ok {
		return "", fmt.Errorf("invalid txid in response: %v", result)
	}

	return txID, nil
}
