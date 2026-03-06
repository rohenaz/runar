//go:build integration

package integration

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"runar-integration/helpers"
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

	artifact, err := helpers.CompileContract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts", map[string]interface{}{
		"pubKey": pubKeyHex,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("SchnorrZKP script: %d bytes", len(artifact.Script)/2)

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
	artifact, err := helpers.CompileContract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts", map[string]interface{}{
		"pubKey": pubKeyHex,
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

	unlockHex := helpers.EncodePushPoint(rx, ry) +
		helpers.EncodePushBigInt(sBad) +
		helpers.EncodePushBigInt(e)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, funder.P2PKHScript(), 49000)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
