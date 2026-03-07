//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	runar "github.com/icellan/runar/packages/runar-go"
)

func deployEscrow(t *testing.T, buyer, seller, arbiter *helpers.Wallet, funder *helpers.Wallet) *runar.RunarContract {
	t.Helper()

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/escrow/Escrow.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile escrow: %v", err)
	}

	contract := runar.NewRunarContract(artifact, []interface{}{
		buyer.PubKeyHex(), seller.PubKeyHex(), arbiter.PubKeyHex(),
	})

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

func spendEscrowMethod(t *testing.T, contract *runar.RunarContract, signer *helpers.Wallet, methodIdx int) string {
	t.Helper()
	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	receiverScript := signer.P2PKHScript()
	spendTx, err := helpers.BuildSpendTx(utxo, receiverScript, 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	sigHex, err := helpers.SignInput(spendTx, 0, signer.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	unlockHex := helpers.EncodePushBytes(sigBytes) + helpers.EncodeMethodIndex(methodIdx)

	spendHex, err := helpers.SpendContract(utxo, unlockHex, receiverScript, 4500)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}
	return spendHex
}

func TestEscrow_Compile(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/escrow/Escrow.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if artifact.ContractName != "Escrow" {
		t.Fatalf("expected contract name Escrow, got %s", artifact.ContractName)
	}
	t.Logf("Escrow compiled: %d bytes", len(artifact.Script)/2)
}

func TestEscrow_DeployThreePubKeys(t *testing.T) {
	buyer := helpers.NewWallet()
	seller := helpers.NewWallet()
	arbiter := helpers.NewWallet()

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/escrow/Escrow.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	contract := runar.NewRunarContract(artifact, []interface{}{
		buyer.PubKeyHex(), seller.PubKeyHex(), arbiter.PubKeyHex(),
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

	txid, _, err := contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 5000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if len(txid) != 64 {
		t.Fatalf("expected 64-char txid, got %d", len(txid))
	}
	t.Logf("deployed with 3 distinct pubkeys: %s", txid)
}

func TestEscrow_DeploySameKey(t *testing.T) {
	wallet := helpers.NewWallet()

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/escrow/Escrow.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	contract := runar.NewRunarContract(artifact, []interface{}{
		wallet.PubKeyHex(), wallet.PubKeyHex(), wallet.PubKeyHex(),
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

	txid, _, err := contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 5000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if len(txid) != 64 {
		t.Fatalf("expected 64-char txid, got %d", len(txid))
	}
	t.Logf("deployed with same key for all roles: %s", txid)
}

func TestEscrow_ReleaseBySeller(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	contract := deployEscrow(t, buyer, seller, arbiter, seller)
	spendHex := spendEscrowMethod(t, contract, seller, 0)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_ReleaseByArbiter(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	contract := deployEscrow(t, buyer, seller, arbiter, arbiter)
	spendHex := spendEscrowMethod(t, contract, arbiter, 1)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_RefundToBuyer(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	contract := deployEscrow(t, buyer, seller, arbiter, buyer)
	spendHex := spendEscrowMethod(t, contract, buyer, 2)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_RefundByArbiter(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	contract := deployEscrow(t, buyer, seller, arbiter, arbiter)
	spendHex := spendEscrowMethod(t, contract, arbiter, 3)
	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestEscrow_WrongSigner_Rejected(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	contract := deployEscrow(t, buyer, seller, arbiter, seller)
	// Method 0 (releaseBySeller) but signed by buyer — should fail
	spendHex := spendEscrowMethod(t, contract, buyer, 0)
	helpers.AssertTxRejected(t, spendHex)
}

func TestEscrow_InvalidMethodIndex_Rejected(t *testing.T) {
	buyer, seller, arbiter := helpers.NewWallet(), helpers.NewWallet(), helpers.NewWallet()
	contract := deployEscrow(t, buyer, seller, arbiter, seller)
	// Method index 5 doesn't exist
	spendHex := spendEscrowMethod(t, contract, seller, 5)
	helpers.AssertTxRejected(t, spendHex)
}
