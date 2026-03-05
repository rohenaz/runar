//go:build integration

package integration

import (
	"encoding/hex"
	"math/big"
	"testing"

	"runar-integration/helpers"
)

func TestOracle_ValidSettle(t *testing.T) {
	receiver := helpers.NewWallet()
	oracleKP, err := helpers.GenerateRabinKeyPair()
	if err != nil {
		t.Fatalf("rabin keygen: %v", err)
	}

	artifact, err := helpers.CompileContract("examples/ts/oracle-price/OraclePriceFeed.runar.ts", map[string]interface{}{
		"oraclePubKey": oracleKP.N,
		"receiver":     receiver.PubKeyHex(),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("OraclePriceFeed script: %d bytes", len(artifact.Script)/2)

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast deploy: %v", err)
	}

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// Oracle signs price=55001 (above 50000 threshold)
	price := int64(55001)
	msgBytes := helpers.Num2binLE(big.NewInt(price), 8)
	rabinSig, err := helpers.RabinSign(msgBytes, oracleKP)
	if err != nil {
		t.Fatalf("rabin sign: %v", err)
	}

	// Build spending TX
	spendTx, err := helpers.BuildSpendTx(contractUTXO, receiver.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	receiverSigHex, err := helpers.SignInput(spendTx, 0, receiver.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	receiverSigBytes, _ := hex.DecodeString(receiverSigHex)

	// Unlocking: <price> <rabinSig> <padding> <receiverSig>
	unlockHex := helpers.EncodePushInt(price) +
		helpers.EncodePushBigInt(rabinSig.Sig) +
		helpers.EncodePushBigInt(rabinSig.Padding) +
		helpers.EncodePushBytes(receiverSigBytes)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, receiver.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	txid := helpers.AssertTxAccepted(t, spendHex)
	helpers.AssertTxInBlock(t, txid)
}

func TestOracle_PriceBelowThreshold_Rejected(t *testing.T) {
	receiver := helpers.NewWallet()
	oracleKP, err := helpers.GenerateRabinKeyPair()
	if err != nil {
		t.Fatalf("rabin keygen: %v", err)
	}

	artifact, err := helpers.CompileContract("examples/ts/oracle-price/OraclePriceFeed.runar.ts", map[string]interface{}{
		"oraclePubKey": oracleKP.N,
		"receiver":     receiver.PubKeyHex(),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, funder)
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

	// Price 49999 is below the 50000 threshold
	price := int64(49999)
	msgBytes := helpers.Num2binLE(big.NewInt(price), 8)
	rabinSig, err := helpers.RabinSign(msgBytes, oracleKP)
	if err != nil {
		t.Fatalf("rabin sign: %v", err)
	}

	spendTx, err := helpers.BuildSpendTx(contractUTXO, receiver.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}
	receiverSigHex, err := helpers.SignInput(spendTx, 0, receiver.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	receiverSigBytes, _ := hex.DecodeString(receiverSigHex)

	unlockHex := helpers.EncodePushInt(price) +
		helpers.EncodePushBigInt(rabinSig.Sig) +
		helpers.EncodePushBigInt(rabinSig.Padding) +
		helpers.EncodePushBytes(receiverSigBytes)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, receiver.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}

func TestOracle_WrongReceiver_Rejected(t *testing.T) {
	receiver := helpers.NewWallet()
	wrongKey := helpers.NewWallet()
	oracleKP, err := helpers.GenerateRabinKeyPair()
	if err != nil {
		t.Fatalf("rabin keygen: %v", err)
	}

	artifact, err := helpers.CompileContract("examples/ts/oracle-price/OraclePriceFeed.runar.ts", map[string]interface{}{
		"oraclePubKey": oracleKP.N,
		"receiver":     receiver.PubKeyHex(),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(artifact.Script, funding, 5000, funder)
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

	price := int64(55001)
	msgBytes := helpers.Num2binLE(big.NewInt(price), 8)
	rabinSig, err := helpers.RabinSign(msgBytes, oracleKP)
	if err != nil {
		t.Fatalf("rabin sign: %v", err)
	}

	// Sign with wrong key (not the receiver)
	spendTx, err := helpers.BuildSpendTx(contractUTXO, wrongKey.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}
	wrongSigHex, err := helpers.SignInput(spendTx, 0, wrongKey.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	wrongSigBytes, _ := hex.DecodeString(wrongSigHex)

	unlockHex := helpers.EncodePushInt(price) +
		helpers.EncodePushBigInt(rabinSig.Sig) +
		helpers.EncodePushBigInt(rabinSig.Padding) +
		helpers.EncodePushBytes(wrongSigBytes)

	spendHex, err := helpers.SpendContract(contractUTXO, unlockHex, wrongKey.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("spend: %v", err)
	}

	helpers.AssertTxRejected(t, spendHex)
}
