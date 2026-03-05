package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
)

var rpcID uint64

func rpcURL() string {
	if u := os.Getenv("RPC_URL"); u != "" {
		return u
	}
	return "http://localhost:18332"
}

func rpcUser() string {
	if u := os.Getenv("RPC_USER"); u != "" {
		return u
	}
	return "bitcoin"
}

func rpcPass() string {
	if p := os.Getenv("RPC_PASS"); p != "" {
		return p
	}
	return "bitcoin"
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint64        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RPCCall makes a JSON-RPC 1.0 call to the Bitcoin node.
func RPCCall(method string, params ...interface{}) (json.RawMessage, error) {
	if params == nil {
		params = []interface{}{}
	}
	reqBody, _ := json.Marshal(rpcRequest{
		JSONRPC: "1.0",
		ID:      atomic.AddUint64(&rpcID, 1),
		Method:  method,
		Params:  params,
	})

	req, err := http.NewRequest("POST", rpcURL(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(rpcUser(), rpcPass())
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RPC connection failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("RPC response parse error: %w (body: %s)", err, string(body))
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

// Mine generates n blocks on the regtest node.
func Mine(n int) error {
	// Try generate first (older nodes), then generatetoaddress
	_, err := RPCCall("generate", n)
	if err != nil {
		addrResult, err2 := RPCCall("getnewaddress")
		if err2 != nil {
			return fmt.Errorf("getnewaddress: %w", err2)
		}
		var addr string
		json.Unmarshal(addrResult, &addr)
		_, err = RPCCall("generatetoaddress", n, addr)
		if err != nil {
			return fmt.Errorf("generatetoaddress: %w", err)
		}
	}
	return nil
}

// SendToAddress sends BTC from the wallet to the given address.
func SendToAddress(addr string, btcAmount float64) (string, error) {
	result, err := RPCCall("sendtoaddress", addr, btcAmount)
	if err != nil {
		return "", err
	}
	var txid string
	json.Unmarshal(result, &txid)
	return txid, nil
}

// SendRawTransaction broadcasts a raw transaction hex.
func SendRawTransaction(txHex string) (string, error) {
	result, err := RPCCall("sendrawtransaction", txHex)
	if err != nil {
		return "", err
	}
	var txid string
	json.Unmarshal(result, &txid)
	return txid, nil
}

// GetRawTransaction fetches a transaction by txid (verbose mode).
func GetRawTransaction(txid string) (map[string]interface{}, error) {
	result, err := RPCCall("getrawtransaction", txid, true)
	if err != nil {
		return nil, err
	}
	var tx map[string]interface{}
	json.Unmarshal(result, &tx)
	return tx, nil
}

// IsNodeAvailable checks if the regtest node is reachable.
func IsNodeAvailable() bool {
	_, err := RPCCall("getblockcount")
	return err == nil
}
