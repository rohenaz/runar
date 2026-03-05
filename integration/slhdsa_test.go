//go:build integration

package integration

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"runar-integration/helpers"
)

const slhdsaTestPK = "00000000000000000000000000000000b618cb38f7f785488c9768f3a2972baf"
const slhdsaTestMsg = "736c682d647361207465737420766563746f72" // "slh-dsa test vector"

func loadSLHDSATestSig(t *testing.T) []byte {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	sigFile := filepath.Join(filepath.Dir(thisFile), "..", "conformance", "testdata", "slhdsa-test-sig.hex")
	data, err := os.ReadFile(sigFile)
	if err != nil {
		t.Skipf("SLH-DSA test signature file not found: %v", err)
	}
	sigHex := strings.TrimSpace(string(data))
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatalf("decode sig hex: %v", err)
	}
	if len(sigBytes) != 7856 {
		t.Fatalf("expected 7856-byte sig, got %d", len(sigBytes))
	}
	return sigBytes
}

func TestSLHDSA_ValidSpend(t *testing.T) {
	if testing.Short() {
		t.Skip("SLH-DSA is very slow (~248 KB script), skipping in short mode")
	}

	sigBytes := loadSLHDSATestSig(t)
	msgBytes, _ := hex.DecodeString(slhdsaTestMsg)

	artifact, err := helpers.CompileContract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts", map[string]interface{}{
		"pubkey": slhdsaTestPK,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("SLH-DSA script: %d bytes", len(artifact.Script)/2)

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 1.0)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 50000, funder)
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

	// Unlocking: <msg> <sig>
	unlockHex := helpers.EncodePushBytes(msgBytes) + helpers.EncodePushBytes(sigBytes)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, funder.P2PKHScript(), 49000)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestSLHDSA_TamperedSig_Rejected(t *testing.T) {
	if testing.Short() {
		t.Skip("SLH-DSA is very slow, skipping in short mode")
	}

	sigBytes := loadSLHDSATestSig(t)
	msgBytes, _ := hex.DecodeString(slhdsaTestMsg)

	// Tamper
	tampered := make([]byte, len(sigBytes))
	copy(tampered, sigBytes)
	tampered[500] ^= 0xff

	artifact, err := helpers.CompileContract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts", map[string]interface{}{
		"pubkey": slhdsaTestPK,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 1.0)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 50000, funder)
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

	unlockHex := helpers.EncodePushBytes(msgBytes) + helpers.EncodePushBytes(tampered)
	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, funder.P2PKHScript(), 49000)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
