//go:build integration

package integration

import (
	"testing"

	"runar-integration/helpers"
)

func TestWOTS_ValidSpend(t *testing.T) {
	if testing.Short() {
		t.Skip("WOTS+ is slow, skipping in short mode")
	}

	// Generate WOTS+ keypair
	seed := make([]byte, 32)
	seed[0] = 0x42
	pubSeed := make([]byte, 32)
	pubSeed[0] = 0x01
	kp := helpers.WOTSKeygen(seed, pubSeed)

	artifact, err := helpers.CompileContract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts", map[string]interface{}{
		"pubkey": helpers.WOTSPubKeyHex(kp),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("WOTS+ script: %d bytes", len(artifact.Script)/2)

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.1)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 10000, funder)
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

	// Sign a message
	msg := []byte("spend this UTXO")
	sig := helpers.WOTSSign(msg, kp.SK, kp.PubSeed)

	// Unlocking script: <msg> <sig>
	unlockHex := helpers.EncodePushBytes(msg) + helpers.EncodePushBytes(sig)

	receiverScript := funder.P2PKHScript()
	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, receiverScript, 9000)
	if err != nil {
		t.Fatalf("spend contract: %v", err)
	}

	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestWOTS_TamperedSig_Rejected(t *testing.T) {
	if testing.Short() {
		t.Skip("WOTS+ is slow, skipping in short mode")
	}

	seed := make([]byte, 32)
	seed[0] = 0x42
	pubSeed := make([]byte, 32)
	pubSeed[0] = 0x01
	kp := helpers.WOTSKeygen(seed, pubSeed)

	artifact, err := helpers.CompileContract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts", map[string]interface{}{
		"pubkey": helpers.WOTSPubKeyHex(kp),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.1)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 10000, funder)
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

	msg := []byte("spend this UTXO")
	sig := helpers.WOTSSign(msg, kp.SK, kp.PubSeed)

	// Tamper with signature
	tampered := make([]byte, len(sig))
	copy(tampered, sig)
	tampered[100] ^= 0xff

	unlockHex := helpers.EncodePushBytes(msg) + helpers.EncodePushBytes(tampered)
	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, funder.P2PKHScript(), 9000)
	if err != nil {
		t.Fatalf("spend contract: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}

func TestWOTS_WrongMessage_Rejected(t *testing.T) {
	if testing.Short() {
		t.Skip("WOTS+ is slow, skipping in short mode")
	}

	seed := make([]byte, 32)
	seed[0] = 0x42
	pubSeed := make([]byte, 32)
	pubSeed[0] = 0x01
	kp := helpers.WOTSKeygen(seed, pubSeed)

	artifact, err := helpers.CompileContract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts", map[string]interface{}{
		"pubkey": helpers.WOTSPubKeyHex(kp),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.1)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 10000, funder)
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

	// Sign one message, try to verify with another
	originalMsg := []byte("original message")
	sig := helpers.WOTSSign(originalMsg, kp.SK, kp.PubSeed)

	wrongMsg := []byte("different message")
	unlockHex := helpers.EncodePushBytes(wrongMsg) + helpers.EncodePushBytes(sig)
	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, funder.P2PKHScript(), 9000)
	if err != nil {
		t.Fatalf("spend contract: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
