//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"
)

func compileEscrow(t *testing.T, buyer, seller, arbiter *helpers.Wallet) string {
	t.Helper()
	artifact, err := helpers.CompileContract("examples/ts/escrow/Escrow.runar.ts", map[string]interface{}{
		"buyer":   buyer.PubKeyHex(),
		"seller":  seller.PubKeyHex(),
		"arbiter": arbiter.PubKeyHex(),
	})
	if err != nil {
		t.Fatalf("compile escrow: %v", err)
	}
	return artifact.Script
}

func deployEscrow(t *testing.T, scriptHex string, funder *helpers.Wallet) *helpers.UTXO {
	t.Helper()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
	deployHex, err := helpers.DeployContract(scriptHex, funding, 5000, funder)
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

func spendEscrowMethod(t *testing.T, contractUTXO *helpers.UTXO, signer *helpers.Wallet, methodIdx int) string {
	t.Helper()
	receiverScript := signer.P2PKHScript()
	spendTx, err := helpers.BuildSpendTx(contractUTXO, receiverScript, 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	sigHex, err := helpers.SignInput(spendTx, 0, signer.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	unlockHex := helpers.EncodePushBytes(sigBytes) + helpers.EncodeMethodIndex(methodIdx)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, receiverScript, 4500)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}
	return spendHex
}

func TestEscrow_ReleaseBySeller(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	scriptHex := compileEscrow(t, buyer, seller, arbiter)
	utxo := deployEscrow(t, scriptHex, seller)
	spendHex := spendEscrowMethod(t, utxo, seller, 0)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_ReleaseByArbiter(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	scriptHex := compileEscrow(t, buyer, seller, arbiter)
	utxo := deployEscrow(t, scriptHex, arbiter)
	spendHex := spendEscrowMethod(t, utxo, arbiter, 1)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_RefundToBuyer(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	scriptHex := compileEscrow(t, buyer, seller, arbiter)
	utxo := deployEscrow(t, scriptHex, buyer)
	spendHex := spendEscrowMethod(t, utxo, buyer, 2)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_RefundByArbiter(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	scriptHex := compileEscrow(t, buyer, seller, arbiter)
	utxo := deployEscrow(t, scriptHex, arbiter)
	spendHex := spendEscrowMethod(t, utxo, arbiter, 3)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_WrongSigner_Rejected(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	scriptHex := compileEscrow(t, buyer, seller, arbiter)
	utxo := deployEscrow(t, scriptHex, seller)
	// Method 0 (releaseBySeller) but signed by buyer — should fail
	spendHex := spendEscrowMethod(t, utxo, buyer, 0)
	helpers.AssertTxRejected(t, spendHex)
}

func TestEscrow_InvalidMethodIndex_Rejected(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	scriptHex := compileEscrow(t, buyer, seller, arbiter)
	utxo := deployEscrow(t, scriptHex, seller)
	// Method index 5 doesn't exist
	spendHex := spendEscrowMethod(t, utxo, seller, 5)
	helpers.AssertTxRejected(t, spendHex)
}
