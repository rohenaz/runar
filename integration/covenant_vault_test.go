//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"
)

func TestCovenantVault_ValidSpend(t *testing.T) {
	owner := helpers.NewWallet()
	recipient := helpers.NewWallet()
	minAmount := int64(1000)

	artifact, err := helpers.CompileContract("examples/ts/covenant-vault/CovenantVault.runar.ts", map[string]interface{}{
		"owner":     owner.PubKeyHex(),
		"recipient": recipient.PubKeyHashHex(),
		"minAmount": float64(minAmount),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("CovenantVault script: %d bytes", len(artifact.Script)/2)

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, funder)
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

	// Build spend TX
	receiverScript := owner.P2PKHScript()
	spendTx, err := helpers.BuildSpendTx(contractUTXO, receiverScript, 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	// Sign with owner's key
	sigHex, err := helpers.SignInput(spendTx, 0, owner.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	// OP_PUSH_TX for checkPreimage
	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}
	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// spend(sig: Sig, amount: bigint, txPreimage: SigHashPreimage)
	// Compiler inserts implicit _opPushTxSig before declared params.
	// Unlocking script order: <opPushTxSig> <sig> <amount> <txPreimage>
	amount := int64(2000) // >= minAmount (1000)
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushInt(amount) +
		helpers.EncodePushBytes(preimageBytes)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	txid := helpers.AssertTxAccepted(t, spendTx.Hex())
	helpers.AssertTxInBlock(t, txid)
}

func TestCovenantVault_BelowMinAmount_Rejected(t *testing.T) {
	owner := helpers.NewWallet()
	recipient := helpers.NewWallet()
	minAmount := int64(1000)

	artifact, err := helpers.CompileContract("examples/ts/covenant-vault/CovenantVault.runar.ts", map[string]interface{}{
		"owner":     owner.PubKeyHex(),
		"recipient": recipient.PubKeyHashHex(),
		"minAmount": float64(minAmount),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, funder)
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

	spendTx, err := helpers.BuildSpendTx(contractUTXO, owner.P2PKHScript(), 4500)
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

	// Amount 500 < minAmount 1000 — should fail assert
	// Unlocking script order: <opPushTxSig> <sig> <amount> <txPreimage>
	amount := int64(500)
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushInt(amount) +
		helpers.EncodePushBytes(preimageBytes)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	helpers.AssertTxRejected(t, spendTx.Hex())
}
