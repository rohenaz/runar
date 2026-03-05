//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"
)

func TestP2PKH_ValidUnlock(t *testing.T) {
	owner := helpers.NewWallet()

	// Compile P2PKH contract with owner's pubKeyHash
	artifact, err := helpers.CompileContract("examples/ts/p2pkh/P2PKH.runar.ts", map[string]interface{}{
		"pubKeyHash": owner.PubKeyHashHex(),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("P2PKH script: %d bytes", len(artifact.Script)/2)

	// Fund wallet and deploy contract
	funding, err := helpers.FundWallet(owner, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, owner)
	if err != nil {
		t.Fatalf("deploy tx: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast deploy: %v", err)
	}
	t.Logf("deployed: %s", deployTxid)

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find contract UTXO: %v", err)
	}

	// Build spending transaction
	receiverScript := owner.P2PKHScript()
	spendTx, err := helpers.BuildSpendTx(contractUTXO, receiverScript, 4500)
	if err != nil {
		t.Fatalf("build spend tx: %v", err)
	}

	// Sign with owner's key
	sigHex, err := helpers.SignInput(spendTx, 0, owner.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	// Build unlocking script: <sig> <pubKey>
	unlockHex := helpers.EncodePushBytes(sigBytes) + helpers.EncodePushBytes(owner.PubKeyBytes)
	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, receiverScript, 4500)
	if err != nil {
		t.Fatalf("spend contract: %v", err)
	}

	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestP2PKH_WrongKey_Rejected(t *testing.T) {
	owner := helpers.NewWallet()
	attacker := helpers.NewWallet()

	artifact, err := helpers.CompileContract("examples/ts/p2pkh/P2PKH.runar.ts", map[string]interface{}{
		"pubKeyHash": owner.PubKeyHashHex(),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funding, err := helpers.FundWallet(attacker, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, attacker)
	if err != nil {
		t.Fatalf("deploy tx: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast deploy: %v", err)
	}

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Build spend signed with attacker's key (wrong key, hash won't match)
	spendTx, err := helpers.BuildSpendTx(contractUTXO, attacker.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}
	sigHex, err := helpers.SignInput(spendTx, 0, attacker.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	unlockHex := helpers.EncodePushBytes(sigBytes) + helpers.EncodePushBytes(attacker.PubKeyBytes)
	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, attacker.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("spend contract: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}

func TestP2PKH_WrongSig_Rejected(t *testing.T) {
	owner := helpers.NewWallet()
	wrongSigner := helpers.NewWallet()

	artifact, err := helpers.CompileContract("examples/ts/p2pkh/P2PKH.runar.ts", map[string]interface{}{
		"pubKeyHash": owner.PubKeyHashHex(),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funding, err := helpers.FundWallet(owner, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, owner)
	if err != nil {
		t.Fatalf("deploy tx: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast deploy: %v", err)
	}

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Correct pubKey but signature from a different private key
	spendTx, err := helpers.BuildSpendTx(contractUTXO, owner.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}
	sigHex, err := helpers.SignInput(spendTx, 0, wrongSigner.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	// Use owner's pubKey but wrongSigner's signature
	unlockHex := helpers.EncodePushBytes(sigBytes) + helpers.EncodePushBytes(owner.PubKeyBytes)
	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, owner.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("spend contract: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
