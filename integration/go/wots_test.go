//go:build integration

package integration

import (
	"testing"

	"runar-integration/helpers"

	runar "github.com/icellan/runar/packages/runar-go"
)

func deployWOTS(t *testing.T, kp helpers.WOTSKeyPair, funder *helpers.Wallet) *runar.RunarContract {
	t.Helper()

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("WOTS+ script: %d bytes", len(artifact.Script)/2)

	contract := runar.NewRunarContract(artifact, []interface{}{helpers.WOTSPubKeyHex(kp)})

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

	_, _, err = contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 10000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}

	return contract
}

func TestWOTS_Compile(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if artifact.ContractName != "PostQuantumWallet" {
		t.Fatalf("expected contract name PostQuantumWallet, got %s", artifact.ContractName)
	}
	t.Logf("WOTS+ compiled: %d bytes", len(artifact.Script)/2)
}

func TestWOTS_ScriptSize(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	scriptBytes := len(artifact.Script) / 2
	if scriptBytes < 5000 || scriptBytes > 50000 {
		t.Fatalf("expected script size 5-50 KB, got %d bytes", scriptBytes)
	}
	t.Logf("WOTS+ script size: %d bytes", scriptBytes)
}

func TestWOTS_Deploy(t *testing.T) {
	seed := make([]byte, 32)
	seed[0] = 0x42
	pubSeed := make([]byte, 32)
	pubSeed[0] = 0x01
	kp := helpers.WOTSKeygen(seed, pubSeed)

	funder := helpers.NewWallet()
	contract := deployWOTS(t, kp, funder)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed WOTS+ contract")
}

func TestWOTS_DeployDifferentSeed(t *testing.T) {
	seed1 := make([]byte, 32)
	seed1[0] = 0xAA
	pubSeed1 := make([]byte, 32)
	pubSeed1[0] = 0x01
	kp1 := helpers.WOTSKeygen(seed1, pubSeed1)

	seed2 := make([]byte, 32)
	seed2[0] = 0xBB
	pubSeed2 := make([]byte, 32)
	pubSeed2[0] = 0x02
	kp2 := helpers.WOTSKeygen(seed2, pubSeed2)

	if helpers.WOTSPubKeyHex(kp1) == helpers.WOTSPubKeyHex(kp2) {
		t.Fatalf("expected different pubkeys from different seeds")
	}

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Deploy with kp1
	contract1 := runar.NewRunarContract(artifact, []interface{}{helpers.WOTSPubKeyHex(kp1)})
	funder1 := helpers.NewWallet()
	helpers.RPCCall("importaddress", funder1.Address, "", false)
	_, err = helpers.FundWallet(funder1, 1.0)
	if err != nil {
		t.Fatalf("fund1: %v", err)
	}
	provider1 := helpers.NewRPCProvider()
	signer1, _ := helpers.SDKSignerFromWallet(funder1)
	txid1, _, err := contract1.Deploy(provider1, signer1, runar.DeployOptions{Satoshis: 10000})
	if err != nil {
		t.Fatalf("deploy1: %v", err)
	}

	// Deploy with kp2
	contract2 := runar.NewRunarContract(artifact, []interface{}{helpers.WOTSPubKeyHex(kp2)})
	funder2 := helpers.NewWallet()
	helpers.RPCCall("importaddress", funder2.Address, "", false)
	_, err = helpers.FundWallet(funder2, 1.0)
	if err != nil {
		t.Fatalf("fund2: %v", err)
	}
	provider2 := helpers.NewRPCProvider()
	signer2, _ := helpers.SDKSignerFromWallet(funder2)
	txid2, _, err := contract2.Deploy(provider2, signer2, runar.DeployOptions{Satoshis: 10000})
	if err != nil {
		t.Fatalf("deploy2: %v", err)
	}

	if txid1 == txid2 {
		t.Fatalf("expected different txids, got same: %s", txid1)
	}
	t.Logf("seed1 txid: %s, seed2 txid: %s", txid1, txid2)
}

func TestWOTS_ValidSpend(t *testing.T) {
	if testing.Short() {
		t.Skip("WOTS+ is slow, skipping in short mode")
	}

	seed := make([]byte, 32)
	seed[0] = 0x42
	pubSeed := make([]byte, 32)
	pubSeed[0] = 0x01
	kp := helpers.WOTSKeygen(seed, pubSeed)

	funder := helpers.NewWallet()
	contract := deployWOTS(t, kp, funder)

	// Sign a message
	msg := []byte("spend this UTXO")
	sig := helpers.WOTSSign(msg, kp.SK, kp.PubSeed)

	// Unlocking script: <msg> <sig>
	unlockHex := helpers.EncodePushBytes(msg) + helpers.EncodePushBytes(sig)

	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	receiverScript := funder.P2PKHScript()
	spendHex, err := helpers.SpendContract(utxo, unlockHex, receiverScript, 9000)
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

	funder := helpers.NewWallet()
	contract := deployWOTS(t, kp, funder)

	msg := []byte("spend this UTXO")
	sig := helpers.WOTSSign(msg, kp.SK, kp.PubSeed)

	// Tamper with signature
	tampered := make([]byte, len(sig))
	copy(tampered, sig)
	tampered[100] ^= 0xff

	unlockHex := helpers.EncodePushBytes(msg) + helpers.EncodePushBytes(tampered)
	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	spendHex, err := helpers.SpendContract(utxo, unlockHex, funder.P2PKHScript(), 9000)
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

	funder := helpers.NewWallet()
	contract := deployWOTS(t, kp, funder)

	originalMsg := []byte("original message")
	sig := helpers.WOTSSign(originalMsg, kp.SK, kp.PubSeed)

	wrongMsg := []byte("different message")
	unlockHex := helpers.EncodePushBytes(wrongMsg) + helpers.EncodePushBytes(sig)
	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	spendHex, err := helpers.SpendContract(utxo, unlockHex, funder.P2PKHScript(), 9000)
	if err != nil {
		t.Fatalf("spend contract: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
