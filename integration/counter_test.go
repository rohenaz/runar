//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

// buildStatefulSpendTx creates a stateful continuation transaction.
// The output carries the same locking script with updated state.
// Returns the transaction (unsigned) for preimage computation.
func buildStatefulSpendTx(contractUTXO *helpers.UTXO, continuationScriptHex string, outputSats int64) (*transaction.Transaction, error) {
	lockScript, err := script.NewFromHex(contractUTXO.Script)
	if err != nil {
		return nil, err
	}
	contScript, err := script.NewFromHex(continuationScriptHex)
	if err != nil {
		return nil, err
	}

	tx := transaction.NewTransaction()
	tx.AddInputWithOutput(&transaction.TransactionInput{
		SourceTXID:       helpers.TxidToChainHash(contractUTXO.Txid),
		SourceTxOutIndex: uint32(contractUTXO.Vout),
		SequenceNumber:   transaction.DefaultSequenceNumber,
	}, &transaction.TransactionOutput{
		Satoshis:      uint64(contractUTXO.Satoshis),
		LockingScript: lockScript,
	})

	// Output 0: continuation with updated state
	tx.AddOutput(&transaction.TransactionOutput{
		Satoshis:      uint64(outputSats),
		LockingScript: contScript,
	})

	return tx, nil
}

// buildContinuationScript replaces the state section (after OP_RETURN) in the
// current locking script with new serialized state bytes.
func buildContinuationScript(currentScriptHex string, stateLen int, newStateHex string) string {
	// Current script = codePart + 6a (OP_RETURN) + currentState
	// We keep codePart + 6a and replace the state
	codeAndOpReturnLen := len(currentScriptHex)/2 - stateLen
	codePart := currentScriptHex[:codeAndOpReturnLen*2]
	return codePart + newStateHex
}

// serializeBigintState serializes a bigint as 8-byte LE (via num2bin) for state.
func serializeBigintState(n int64) string {
	// 8-byte little-endian encoding matching OP_NUM2BIN behavior
	buf := make([]byte, 8)
	v := n
	negative := v < 0
	if negative {
		v = -v
	}
	for i := 0; i < 7; i++ {
		buf[i] = byte(v & 0xff)
		v >>= 8
	}
	if negative {
		buf[7] = 0x80
	}
	return hex.EncodeToString(buf)
}

func TestCounter_Increment(t *testing.T) {
	// Compile Counter with count=0
	artifact, err := helpers.CompileContract("examples/ts/stateful-counter/Counter.runar.ts", map[string]interface{}{
		"count": float64(0),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("Counter script: %d bytes", len(artifact.Script)/2)

	// Deploy: code + OP_RETURN + initial state. The computeStateOutputHash
	// extracts the code portion from the preimage's scriptCode field.
	initialState := serializeBigintState(0) // count=0
	deployScript := artifact.Script + "6a" + initialState

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast deploy: %v", err)
	}
	t.Logf("deployed: %s", deployTxid)

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Build continuation: count goes from 0 → 1
	newState := serializeBigintState(1) // count=1
	continuationScript := buildContinuationScript(contractUTXO.Script, 8, newState)

	spendTx, err := buildStatefulSpendTx(contractUTXO, continuationScript, contractUTXO.Satoshis)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	// Compute OP_PUSH_TX signature and preimage
	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}

	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// Unlocking: <_opPushTxSig> <txPreimage> <methodIndex=0>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(0) // increment

	// Set unlocking script on the spending TX
	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	spendTxid, err := helpers.BroadcastAndMine(spendTx.Hex())
	if err != nil {
		t.Fatalf("spend: %v", err)
	}
	t.Logf("increment TX confirmed: %s", spendTxid)
}

// spendStatefulCounter builds and broadcasts one stateful spend for the Counter contract.
// Returns the new UTXO after the spend.
func spendStatefulCounter(t *testing.T, contractUTXO *helpers.UTXO, newCount int64, methodIdx int) *helpers.UTXO {
	t.Helper()
	newState := serializeBigintState(newCount)
	continuationScript := buildContinuationScript(contractUTXO.Script, 8, newState)

	spendTx, err := buildStatefulSpendTx(contractUTXO, continuationScript, contractUTXO.Satoshis)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}

	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(methodIdx)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	spendTxid, err := helpers.BroadcastAndMine(spendTx.Hex())
	if err != nil {
		t.Fatalf("spend (count→%d): %v", newCount, err)
	}
	t.Logf("count→%d TX: %s", newCount, spendTxid)

	utxo, err := helpers.FindUTXOByIndex(spendTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO after spend: %v", err)
	}
	return utxo
}

func TestCounter_IncrementChain(t *testing.T) {
	// Deploy count=0, then increment twice: 0→1→2
	artifact, err := helpers.CompileContract("examples/ts/stateful-counter/Counter.runar.ts", map[string]interface{}{
		"count": float64(0),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	initialState := serializeBigintState(0)
	deployScript := artifact.Script + "6a" + initialState

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	utxo, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Increment 0→1
	utxo = spendStatefulCounter(t, utxo, 1, 0)

	// Increment 1→2
	utxo = spendStatefulCounter(t, utxo, 2, 0)
	_ = utxo
	t.Logf("chain: 0→1→2 succeeded")
}

func TestCounter_IncrementThenDecrement(t *testing.T) {
	// Deploy count=0, increment to 1, then decrement back to 0
	artifact, err := helpers.CompileContract("examples/ts/stateful-counter/Counter.runar.ts", map[string]interface{}{
		"count": float64(0),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	initialState := serializeBigintState(0)
	deployScript := artifact.Script + "6a" + initialState

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	utxo, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Increment 0→1
	utxo = spendStatefulCounter(t, utxo, 1, 0)

	// Decrement 1→0 (method index 1)
	utxo = spendStatefulCounter(t, utxo, 0, 1)
	_ = utxo
	t.Logf("chain: 0→1→0 succeeded")
}

func TestCounter_WrongStateHash_Rejected(t *testing.T) {
	// Deploy count=0, try to increment but claim count=99 in continuation
	artifact, err := helpers.CompileContract("examples/ts/stateful-counter/Counter.runar.ts", map[string]interface{}{
		"count": float64(0),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	initialState := serializeBigintState(0)
	deployScript := artifact.Script + "6a" + initialState

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Build continuation with WRONG state (count=99 instead of 1)
	wrongState := serializeBigintState(99)
	continuationScript := buildContinuationScript(contractUTXO.Script, 8, wrongState)

	spendTx, err := buildStatefulSpendTx(contractUTXO, continuationScript, contractUTXO.Satoshis)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}

	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(0) // increment

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	// The script computes hash(amount + varint + code + OP_RETURN + count=1) but
	// the TX output has count=99, so hashOutputs won't match → rejected
	helpers.AssertTxRejected(t, spendTx.Hex())
}

func TestCounter_DecrementFromZero_Rejected(t *testing.T) {
	// Deploy Counter with count=0 and try to decrement → assert(count > 0) fails
	artifact, err := helpers.CompileContract("examples/ts/stateful-counter/Counter.runar.ts", map[string]interface{}{
		"count": float64(0),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	initialState := serializeBigintState(0)
	deployScript := artifact.Script + "6a" + initialState

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Try decrement: count=0, assert(count > 0) should fail.
	// Even though the assertion will fail before reaching state hash check,
	// we still need valid OP_PUSH_TX args for the script to get that far.
	// count after decrement would be -1 (but assertion stops it)
	newState := serializeBigintState(-1)
	continuationScript := buildContinuationScript(contractUTXO.Script, 8, newState)

	spendTx, err := buildStatefulSpendTx(contractUTXO, continuationScript, contractUTXO.Satoshis)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}

	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// Unlocking: <_opPushTxSig> <txPreimage> <methodIndex=1>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(1) // decrement

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	helpers.AssertTxRejected(t, spendTx.Hex())
}
