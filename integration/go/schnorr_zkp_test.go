//go:build integration

package integration

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"runar-integration/helpers"

	runar "github.com/icellan/runar/packages/runar-go"
)

// secp256k1 curve order
var ecN, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)

// secp256k1 generator point
var ecGx, _ = new(big.Int).SetString("79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798", 16)
var ecGy, _ = new(big.Int).SetString("483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8", 16)

// secp256k1 field prime
var ecP, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f", 16)

// ecMul performs scalar multiplication on secp256k1 using the standard double-and-add algorithm.
func ecMul(px, py, k *big.Int) (*big.Int, *big.Int) {
	rx, ry := new(big.Int), new(big.Int)
	first := true
	for i := k.BitLen() - 1; i >= 0; i-- {
		if !first {
			rx, ry = ecDouble(rx, ry)
		}
		if k.Bit(i) == 1 {
			if first {
				rx.Set(px)
				ry.Set(py)
				first = false
			} else {
				rx, ry = ecAddPoints(rx, ry, px, py)
			}
		}
	}
	return rx, ry
}

func ecDouble(px, py *big.Int) (*big.Int, *big.Int) {
	// lambda = (3*px^2) / (2*py) mod p
	num := new(big.Int).Mul(px, px)
	num.Mul(num, big.NewInt(3))
	num.Mod(num, ecP)
	den := new(big.Int).Mul(big.NewInt(2), py)
	den.ModInverse(den, ecP)
	lambda := new(big.Int).Mul(num, den)
	lambda.Mod(lambda, ecP)

	rx := new(big.Int).Mul(lambda, lambda)
	rx.Sub(rx, px)
	rx.Sub(rx, px)
	rx.Mod(rx, ecP)

	ry := new(big.Int).Sub(px, rx)
	ry.Mul(lambda, ry)
	ry.Sub(ry, py)
	ry.Mod(ry, ecP)
	return rx, ry
}

func ecAddPoints(px, py, qx, qy *big.Int) (*big.Int, *big.Int) {
	if px.Cmp(qx) == 0 && py.Cmp(qy) == 0 {
		return ecDouble(px, py)
	}
	// lambda = (qy - py) / (qx - px) mod p
	num := new(big.Int).Sub(qy, py)
	num.Mod(num, ecP)
	den := new(big.Int).Sub(qx, px)
	den.ModInverse(den, ecP)
	lambda := new(big.Int).Mul(num, den)
	lambda.Mod(lambda, ecP)

	rx := new(big.Int).Mul(lambda, lambda)
	rx.Sub(rx, px)
	rx.Sub(rx, qx)
	rx.Mod(rx, ecP)

	ry := new(big.Int).Sub(px, rx)
	ry.Mul(lambda, ry)
	ry.Sub(ry, py)
	ry.Mod(ry, ecP)
	return rx, ry
}

func TestSchnorr_Compile(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/schnorr-zkp/SchnorrZKP.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if artifact.ContractName != "SchnorrZKP" {
		t.Fatalf("expected contract name SchnorrZKP, got %s", artifact.ContractName)
	}
	t.Logf("SchnorrZKP compiled: %d bytes", len(artifact.Script)/2)
}

func TestSchnorr_ScriptSize(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/schnorr-zkp/SchnorrZKP.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	scriptBytes := len(artifact.Script) / 2
	if scriptBytes < 100000 || scriptBytes > 2000000 {
		t.Fatalf("expected script size 100KB-2MB, got %d bytes", scriptBytes)
	}
	t.Logf("SchnorrZKP script size: %d bytes", scriptBytes)
}

func TestSchnorr_Deploy(t *testing.T) {
	k, _ := rand.Int(rand.Reader, ecN)
	k.Add(k, big.NewInt(1))
	if k.Cmp(ecN) >= 0 {
		k.Sub(k, big.NewInt(1))
	}
	px, py := ecMul(ecGx, ecGy, k)
	pubKeyHex := fmt.Sprintf("%064x%064x", px, py)

	funder := helpers.NewWallet()
	contract := deploySchnorrZKP(t, pubKeyHex, funder)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed SchnorrZKP")
}

func TestSchnorr_DeployDifferentKey(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/schnorr-zkp/SchnorrZKP.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	k1, _ := rand.Int(rand.Reader, ecN)
	k1.Add(k1, big.NewInt(1))
	px1, py1 := ecMul(ecGx, ecGy, k1)
	pk1Hex := fmt.Sprintf("%064x%064x", px1, py1)

	k2, _ := rand.Int(rand.Reader, ecN)
	k2.Add(k2, big.NewInt(1))
	px2, py2 := ecMul(ecGx, ecGy, k2)
	pk2Hex := fmt.Sprintf("%064x%064x", px2, py2)

	// Deploy with key1
	contract1 := runar.NewRunarContract(artifact, []interface{}{pk1Hex})
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

	// Deploy with key2
	contract2 := runar.NewRunarContract(artifact, []interface{}{pk2Hex})
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
	t.Logf("key1 txid: %s, key2 txid: %s", txid1, txid2)
}

func deploySchnorrZKP(t *testing.T, pubKeyHex string, funder *helpers.Wallet) *runar.RunarContract {
	t.Helper()

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/schnorr-zkp/SchnorrZKP.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("SchnorrZKP script: %d bytes", len(artifact.Script)/2)

	contract := runar.NewRunarContract(artifact, []interface{}{pubKeyHex})

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

func TestSchnorr_ValidProof(t *testing.T) {
	if testing.Short() {
		t.Skip("Schnorr EC math is slow, skipping in short mode")
	}

	// Generate keypair and proof parameters.
	// The ecMul codegen uses k+3n trick so any scalar in [1, n-1] works.
	k, _ := rand.Int(rand.Reader, ecN)
	k.Add(k, big.NewInt(1)) // ensure k >= 1
	if k.Cmp(ecN) >= 0 {
		k.Sub(k, big.NewInt(1))
	}
	px, py := ecMul(ecGx, ecGy, k)

	r2, _ := rand.Int(rand.Reader, ecN)
	r2.Add(r2, big.NewInt(1))
	if r2.Cmp(ecN) >= 0 {
		r2.Sub(r2, big.NewInt(1))
	}
	rx, ry := ecMul(ecGx, ecGy, r2)

	e := big.NewInt(12345)

	// s = r + e*k mod n
	s := new(big.Int).Mul(e, k)
	s.Add(s, r2)
	s.Mod(s, ecN)

	// pubKey as 64-byte point (x[32]||y[32])
	pubKeyHex := fmt.Sprintf("%064x%064x", px, py)

	funder := helpers.NewWallet()
	contract := deploySchnorrZKP(t, pubKeyHex, funder)

	// Get the deployed UTXO from the SDK contract
	contractUTXO := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	if contractUTXO == nil {
		t.Fatalf("no current UTXO after deploy")
	}

	// Unlocking: <rPoint> <s> <e>
	unlockHex := helpers.EncodePushPoint(rx, ry) +
		helpers.EncodePushBigInt(s) +
		helpers.EncodePushBigInt(e)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, funder.P2PKHScript(), 49000)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestSchnorr_InvalidS_Rejected(t *testing.T) {
	if testing.Short() {
		t.Skip("Schnorr EC math is slow, skipping in short mode")
	}

	k, _ := rand.Int(rand.Reader, new(big.Int).Sub(ecN, big.NewInt(2)))
	k.Add(k, big.NewInt(1))
	px, py := ecMul(ecGx, ecGy, k)

	r, _ := rand.Int(rand.Reader, new(big.Int).Sub(ecN, big.NewInt(2)))
	r.Add(r, big.NewInt(1))
	rx, ry := ecMul(ecGx, ecGy, r)

	e := big.NewInt(12345)
	s := new(big.Int).Mul(e, k)
	s.Add(s, r)
	s.Mod(s, ecN)

	// Tamper with s
	sBad := new(big.Int).Add(s, big.NewInt(1))
	sBad.Mod(sBad, ecN)

	pubKeyHex := fmt.Sprintf("%064x%064x", px, py)

	funder := helpers.NewWallet()
	contract := deploySchnorrZKP(t, pubKeyHex, funder)

	// Get the deployed UTXO from the SDK contract
	contractUTXO := helpers.SDKUtxoToHelper(contract.GetCurrentUtxo())
	if contractUTXO == nil {
		t.Fatalf("no current UTXO after deploy")
	}

	unlockHex := helpers.EncodePushPoint(rx, ry) +
		helpers.EncodePushBigInt(sBad) +
		helpers.EncodePushBigInt(e)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, funder.P2PKHScript(), 49000)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
