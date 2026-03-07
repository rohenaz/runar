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

	runar "github.com/icellan/runar/packages/runar-go"
)

const slhdsaTestPK = "00000000000000000000000000000000b618cb38f7f785488c9768f3a2972baf"
const slhdsaTestMsg = "736c682d647361207465737420766563746f72" // "slh-dsa test vector"

func loadSLHDSATestSig(t *testing.T) []byte {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	sigFile := filepath.Join(filepath.Dir(thisFile), "..", "..", "conformance", "testdata", "slhdsa-test-sig.hex")
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

func TestSLHDSA_Compile(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if artifact.ContractName != "SPHINCSWallet" {
		t.Fatalf("expected contract name SPHINCSWallet, got %s", artifact.ContractName)
	}
	t.Logf("SLH-DSA compiled: %d bytes", len(artifact.Script)/2)
}

func TestSLHDSA_ScriptSize(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	scriptBytes := len(artifact.Script) / 2
	if scriptBytes < 100000 || scriptBytes > 500000 {
		t.Fatalf("expected script size 100-500 KB, got %d bytes", scriptBytes)
	}
	t.Logf("SLH-DSA script size: %d bytes", scriptBytes)
}

func TestSLHDSA_Deploy(t *testing.T) {
	funder := helpers.NewWallet()
	contract := deploySLHDSA(t, funder)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed SLH-DSA contract")
}

func TestSLHDSA_DeployDifferentKey(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	pk1 := slhdsaTestPK
	pk2 := "11111111111111111111111111111111aaaabbbbccccddddeeeeffff00001111"

	// Deploy with pk1
	contract1 := runar.NewRunarContract(artifact, []interface{}{pk1})
	funder1 := helpers.NewWallet()
	helpers.RPCCall("importaddress", funder1.Address, "", false)
	_, err = helpers.FundWallet(funder1, 1.0)
	if err != nil {
		t.Fatalf("fund1: %v", err)
	}
	provider1 := helpers.NewRPCProvider()
	signer1, _ := helpers.SDKSignerFromWallet(funder1)
	txid1, _, err := contract1.Deploy(provider1, signer1, runar.DeployOptions{Satoshis: 50000})
	if err != nil {
		t.Fatalf("deploy1: %v", err)
	}

	// Deploy with pk2
	contract2 := runar.NewRunarContract(artifact, []interface{}{pk2})
	funder2 := helpers.NewWallet()
	helpers.RPCCall("importaddress", funder2.Address, "", false)
	_, err = helpers.FundWallet(funder2, 1.0)
	if err != nil {
		t.Fatalf("fund2: %v", err)
	}
	provider2 := helpers.NewRPCProvider()
	signer2, _ := helpers.SDKSignerFromWallet(funder2)
	txid2, _, err := contract2.Deploy(provider2, signer2, runar.DeployOptions{Satoshis: 50000})
	if err != nil {
		t.Fatalf("deploy2: %v", err)
	}

	if txid1 == txid2 {
		t.Fatalf("expected different txids, got same: %s", txid1)
	}
	t.Logf("pk1 txid: %s, pk2 txid: %s", txid1, txid2)
}

func deploySLHDSA(t *testing.T, funder *helpers.Wallet) *runar.RunarContract {
	t.Helper()

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("SLH-DSA script: %d bytes", len(artifact.Script)/2)

	contract := runar.NewRunarContract(artifact, []interface{}{slhdsaTestPK})

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

	_, _, err = contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 50000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}

	return contract
}

func TestSLHDSA_ValidSpend(t *testing.T) {
	if testing.Short() {
		t.Skip("SLH-DSA is very slow (~248 KB script), skipping in short mode")
	}

	sigBytes := loadSLHDSATestSig(t)
	msgBytes, _ := hex.DecodeString(slhdsaTestMsg)

	funder := helpers.NewWallet()
	contract := deploySLHDSA(t, funder)

	// Unlocking: <msg> <sig>
	unlockHex := helpers.EncodePushBytes(msgBytes) + helpers.EncodePushBytes(sigBytes)

	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	spendHex, err := helpers.SpendContract(utxo, unlockHex, funder.P2PKHScript(), 49000)
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

	funder := helpers.NewWallet()
	contract := deploySLHDSA(t, funder)

	unlockHex := helpers.EncodePushBytes(msgBytes) + helpers.EncodePushBytes(tampered)

	utxo := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	spendHex, err := helpers.SpendContract(utxo, unlockHex, funder.P2PKHScript(), 49000)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
