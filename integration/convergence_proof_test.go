//go:build integration

package integration

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"runar-integration/helpers"

	runar "github.com/icellan/runar/packages/runar-go"
)

func TestConvergenceProof_ValidProof(t *testing.T) {
	// ConvergenceProof: SmartContract with EC point operations
	// proveConvergence(deltaO: bigint) — proves rA - rB = deltaO * G

	// Pick random scalars a and b, compute rA = a*G, rB = b*G
	a, _ := rand.Int(rand.Reader, ecN)
	b, _ := rand.Int(rand.Reader, ecN)
	deltaO := new(big.Int).Sub(a, b)
	deltaO.Mod(deltaO, ecN)

	rAx, rAy := ecMul(ecGx, ecGy, a)
	rBx, rBy := ecMul(ecGx, ecGy, b)

	// Encode as 64-byte Point hex strings for constructor
	rAHex := fmt.Sprintf("%064x%064x", rAx, rAy)
	rBHex := fmt.Sprintf("%064x%064x", rBx, rBy)

	constructorArgs := map[string]interface{}{
		"rA": rAHex,
		"rB": rBHex,
	}

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/convergence-proof/ConvergenceProof.runar.ts",
		constructorArgs,
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("ConvergenceProof script: %d bytes", len(artifact.Script)/2)

	// Create contract with constructor args in declaration order: rA, rB
	contract := runar.NewRunarContract(artifact, []interface{}{rAHex, rBHex})

	// Fund a wallet and set up provider + signer
	wallet := helpers.NewWallet()
	// Import the wallet address so the node can list its UTXOs
	helpers.RPCCall("importaddress", wallet.Address, "", false)
	_, err = helpers.FundWallet(wallet, 1.0)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	provider := helpers.NewRPCProvider()
	signer, err := helpers.SDKSignerFromWallet(wallet)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	// Deploy
	deployTxid, _, err := contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 500000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	t.Logf("deployed: %s", deployTxid)

	// Call proveConvergence(deltaO) — stateless contract, no OP_PUSH_TX
	// deltaO is a *big.Int; the SDK's encodeArg handles *big.Int natively
	callTxid, _, err := contract.Call("proveConvergence", []interface{}{deltaO}, provider, signer, nil)
	if err != nil {
		t.Fatalf("call proveConvergence: %v", err)
	}
	t.Logf("proveConvergence TX confirmed: %s", callTxid)
}

func TestConvergenceProof_WrongDelta_Rejected(t *testing.T) {
	a, _ := rand.Int(rand.Reader, ecN)
	b, _ := rand.Int(rand.Reader, ecN)

	rAx, rAy := ecMul(ecGx, ecGy, a)
	rBx, rBy := ecMul(ecGx, ecGy, b)

	rAHex := fmt.Sprintf("%064x%064x", rAx, rAy)
	rBHex := fmt.Sprintf("%064x%064x", rBx, rBy)

	constructorArgs := map[string]interface{}{
		"rA": rAHex,
		"rB": rBHex,
	}

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/convergence-proof/ConvergenceProof.runar.ts",
		constructorArgs,
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	contract := runar.NewRunarContract(artifact, []interface{}{rAHex, rBHex})

	wallet := helpers.NewWallet()
	helpers.RPCCall("importaddress", wallet.Address, "", false)
	_, err = helpers.FundWallet(wallet, 1.0)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	provider := helpers.NewRPCProvider()
	signer, err := helpers.SDKSignerFromWallet(wallet)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	_, _, err = contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 500000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}

	// Wrong deltaO — should be rejected
	wrongDelta := big.NewInt(42)
	wrongDeltaHex := bigIntToScriptNumHex(wrongDelta)

	_, _, err = contract.Call("proveConvergence", []interface{}{wrongDeltaHex}, provider, signer, nil)
	if err == nil {
		t.Fatalf("expected call with wrong delta to be rejected, but it succeeded")
	}
	t.Logf("correctly rejected: %v", err)
}

// bigIntToScriptNumHex encodes a *big.Int as a Bitcoin Script number hex string.
// The SDK's encodeArg treats strings as hex-encoded push data, so we produce
// a script-number-encoded hex string (little-endian, sign bit in MSB).
func bigIntToScriptNumHex(n *big.Int) string {
	if n.Sign() == 0 {
		return "" // empty string → OP_0
	}

	abs := new(big.Int).Abs(n)
	absBytes := abs.Bytes() // big-endian
	// Convert to little-endian
	le := make([]byte, len(absBytes))
	for i, b := range absBytes {
		le[len(absBytes)-1-i] = b
	}

	// Add sign bit
	if le[len(le)-1]&0x80 != 0 {
		if n.Sign() < 0 {
			le = append(le, 0x80)
		} else {
			le = append(le, 0x00)
		}
	} else if n.Sign() < 0 {
		le[len(le)-1] |= 0x80
	}

	return hex.EncodeToString(le)
}
