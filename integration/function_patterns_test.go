//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"
)

func deployFunctionPatterns(t *testing.T, owner *helpers.Wallet, initialBalance int64) *helpers.UTXO {
	t.Helper()
	artifact, err := helpers.CompileContract("examples/ts/function-patterns/FunctionPatterns.runar.ts", map[string]interface{}{
		"owner":   owner.PubKeyHex(),
		"balance": float64(initialBalance),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("FunctionPatterns script: %d bytes", len(artifact.Script)/2)

	// Mutable state: balance (8 bytes). owner is readonly (in code part via constructor slot).
	state := serializeBigintState(initialBalance)
	deployScript := artifact.Script + "6a" + state

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
	deployHex, err := helpers.DeployContract(deployScript, funding, 10000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	txid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	utxo, err := helpers.FindUTXOByIndex(txid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}
	return utxo
}

func spendFunctionPatterns(t *testing.T, utxo *helpers.UTXO, owner *helpers.Wallet, newBalance int64, methodIdx int, extraArgs string) *helpers.UTXO {
	t.Helper()
	newState := serializeBigintState(newBalance)
	continuationScript := buildContinuationScript(utxo.Script, 8, newState)

	spendTx, err := buildStatefulSpendTx(utxo, continuationScript, utxo.Satoshis)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	sigHex, err := helpers.SignInput(spendTx, 0, owner.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}
	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// Unlocking: <opPushTxSig> <sig> <extraArgs> <txPreimage> <methodIndex>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		extraArgs +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(methodIdx)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	txid, err := helpers.BroadcastAndMine(spendTx.Hex())
	if err != nil {
		t.Fatalf("spend (method %d, balance→%d): %v", methodIdx, newBalance, err)
	}
	t.Logf("balance→%d TX: %s", newBalance, txid)

	newUTXO, err := helpers.FindUTXOByIndex(txid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}
	return newUTXO
}

func TestFunctionPatterns_Deposit(t *testing.T) {
	owner := helpers.NewWallet()
	utxo := deployFunctionPatterns(t, owner, 100)

	// deposit(sig, amount=50) → balance = 150
	// deposit is method 0
	_ = spendFunctionPatterns(t, utxo, owner, 150, 0, helpers.EncodePushInt(50))
}

func TestFunctionPatterns_DepositThenWithdraw(t *testing.T) {
	owner := helpers.NewWallet()
	utxo := deployFunctionPatterns(t, owner, 1000)

	// deposit 500 → 1500
	utxo = spendFunctionPatterns(t, utxo, owner, 1500, 0, helpers.EncodePushInt(500))

	// withdraw(sig, amount=200, feeBps=100) → fee=2, total=202, balance=1298
	// withdraw is method 1
	_ = spendFunctionPatterns(t, utxo, owner, 1298, 1,
		helpers.EncodePushInt(200)+helpers.EncodePushInt(100))
	t.Logf("chain: 1000→1500→1298 succeeded")
}

func TestFunctionPatterns_WrongOwner_Rejected(t *testing.T) {
	owner := helpers.NewWallet()
	attacker := helpers.NewWallet()
	utxo := deployFunctionPatterns(t, owner, 100)

	// Attacker tries to deposit — requireOwner(sig) should fail
	newState := serializeBigintState(200)
	continuationScript := buildContinuationScript(utxo.Script, 8, newState)

	spendTx, err := buildStatefulSpendTx(utxo, continuationScript, utxo.Satoshis)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	sigHex, err := helpers.SignInput(spendTx, 0, attacker.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}
	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// Unlocking: <opPushTxSig> <sig> <amount> <txPreimage> <methodIndex>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushInt(100) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(0)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	helpers.AssertTxRejected(t, spendTx.Hex())
}
