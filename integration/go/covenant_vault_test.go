//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"

	runar "github.com/icellan/runar/packages/runar-go"
)

func deployCovenantVault(t *testing.T, owner, recipient *helpers.Wallet, minAmount int64) *runar.RunarContract {
	t.Helper()

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/covenant-vault/CovenantVault.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("CovenantVault script: %d bytes", len(artifact.Script)/2)

	contract := runar.NewRunarContract(artifact, []interface{}{
		owner.PubKeyHex(),
		recipient.PubKeyHashHex(),
		int64(minAmount),
	})

	funder := helpers.NewWallet()
	helpers.RPCCall("importaddress", funder.Address, "", false)
	_, err = helpers.FundWallet(funder, 1.0)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	provider := helpers.NewRPCProvider()
	signer, err := helpers.SDKSignerFromWallet(funder)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	_, _, err = contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 5000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}

	return contract
}

func TestCovenantVault_Compile(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/covenant-vault/CovenantVault.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if artifact.ContractName != "CovenantVault" {
		t.Fatalf("expected contract name CovenantVault, got %s", artifact.ContractName)
	}
	t.Logf("CovenantVault compiled: %d bytes", len(artifact.Script)/2)
}

func TestCovenantVault_Deploy(t *testing.T) {
	owner := helpers.NewWallet()
	recipient := helpers.NewWallet()
	contract := deployCovenantVault(t, owner, recipient, 1000)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed with minAmount=1000")
}

func TestCovenantVault_DeployZeroMinAmount(t *testing.T) {
	owner := helpers.NewWallet()
	recipient := helpers.NewWallet()
	contract := deployCovenantVault(t, owner, recipient, 0)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed with minAmount=0")
}

func TestCovenantVault_DeployLargeMinAmount(t *testing.T) {
	owner := helpers.NewWallet()
	recipient := helpers.NewWallet()
	contract := deployCovenantVault(t, owner, recipient, 100000000)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed with minAmount=100000000 (1 BTC)")
}

func TestCovenantVault_DeploySameKey(t *testing.T) {
	wallet := helpers.NewWallet()
	contract := deployCovenantVault(t, wallet, wallet, 1000)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed with same key as owner and recipient")
}

func TestCovenantVault_ValidSpend(t *testing.T) {
	owner := helpers.NewWallet()
	recipient := helpers.NewWallet()
	minAmount := int64(1000)

	contract := deployCovenantVault(t, owner, recipient, minAmount)

	// Get UTXO from SDK contract, convert for raw spending
	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())

	// Build spend TX
	receiverScript := owner.P2PKHScript()
	spendTx, err := helpers.BuildSpendTx(utxo, receiverScript, 4500)
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

	contract := deployCovenantVault(t, owner, recipient, minAmount)

	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())

	spendTx, err := helpers.BuildSpendTx(utxo, owner.P2PKHScript(), 4500)
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
