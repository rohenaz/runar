//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

func TestNFT_Transfer(t *testing.T) {
	// SimpleNFT: StatefulSmartContract with addOutput
	// transfer(sig, newOwner, outputSatoshis) — transfers NFT to new owner
	owner := helpers.NewWallet()
	newOwner := helpers.NewWallet()
	tokenIdHex := hex.EncodeToString([]byte("NFT-001"))
	metadataHex := hex.EncodeToString([]byte("My First NFT"))

	artifact, err := helpers.CompileContract("examples/ts/token-nft/NFTExample.runar.ts", map[string]interface{}{
		"owner":    owner.PubKeyHex(),
		"tokenId":  tokenIdHex,
		"metadata": metadataHex,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("SimpleNFT script: %d bytes", len(artifact.Script)/2)

	// Mutable state: owner(33 bytes PubKey)
	// tokenId and metadata are readonly (in code via constructor slots)
	state := hex.EncodeToString(owner.PubKeyBytes)
	deployScript := artifact.Script + "6a" + state

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	deployTxid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	t.Logf("deployed: %s", deployTxid)

	contractUTXO, err := helpers.FindUTXOByIndex(deployTxid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}

	// transfer(sig, newOwner, outputSatoshis) creates one output with new owner
	newState := hex.EncodeToString(newOwner.PubKeyBytes)
	continuationScript := buildContinuationScript(contractUTXO.Script, 33, newState)
	outputSatoshis := int64(4500)

	lockScript, _ := script.NewFromHex(contractUTXO.Script)
	contScript, _ := script.NewFromHex(continuationScript)

	spendTx := transaction.NewTransaction()
	spendTx.AddInputWithOutput(&transaction.TransactionInput{
		SourceTXID:       helpers.TxidToChainHash(contractUTXO.Txid),
		SourceTxOutIndex: uint32(contractUTXO.Vout),
		SequenceNumber:   transaction.DefaultSequenceNumber,
	}, &transaction.TransactionOutput{
		Satoshis:      uint64(contractUTXO.Satoshis),
		LockingScript: lockScript,
	})
	spendTx.AddOutput(&transaction.TransactionOutput{
		Satoshis:      uint64(outputSatoshis),
		LockingScript: contScript,
	})

	sigHex, err := helpers.SignInput(spendTx, 0, owner.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}
	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// transfer is method 0, burn is method 1
	// Unlocking: <opPushTxSig> <sig> <newOwner:PubKey> <outputSatoshis> <txPreimage> <methodIdx>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushBytes(newOwner.PubKeyBytes) +
		helpers.EncodePushInt(outputSatoshis) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(0) // transfer

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	txid := helpers.AssertTxAccepted(t, spendTx.Hex())
	helpers.AssertTxInBlock(t, txid)
}

func TestNFT_Burn(t *testing.T) {
	// burn(sig) — destroys the NFT. No continuation output, just pays to a P2PKH.
	owner := helpers.NewWallet()
	tokenIdHex := hex.EncodeToString([]byte("NFT-BURN-001"))
	metadataHex := hex.EncodeToString([]byte("Burnable NFT"))

	artifact, err := helpers.CompileContract("examples/ts/token-nft/NFTExample.runar.ts", map[string]interface{}{
		"owner":    owner.PubKeyHex(),
		"tokenId":  tokenIdHex,
		"metadata": metadataHex,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	state := hex.EncodeToString(owner.PubKeyBytes)
	deployScript := artifact.Script + "6a" + state

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
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

	// burn(sig) — method 1. The script just verifies checkSig and doesn't
	// check the output hash (no addOutput calls), so a plain spend works.
	spendTx, err := helpers.BuildSpendTx(contractUTXO, owner.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	sigHex, err := helpers.SignInput(spendTx, 0, owner.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}
	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// burn is method 1
	// Unlocking: <opPushTxSig> <sig> <txPreimage> <methodIndex=1>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(1) // burn

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	txid := helpers.AssertTxAccepted(t, spendTx.Hex())
	helpers.AssertTxInBlock(t, txid)
}

func TestNFT_WrongOwner_Rejected(t *testing.T) {
	owner := helpers.NewWallet()
	attacker := helpers.NewWallet()
	tokenIdHex := hex.EncodeToString([]byte("NFT-STEAL-001"))
	metadataHex := hex.EncodeToString([]byte("Steal attempt"))

	artifact, err := helpers.CompileContract("examples/ts/token-nft/NFTExample.runar.ts", map[string]interface{}{
		"owner":    owner.PubKeyHex(),
		"tokenId":  tokenIdHex,
		"metadata": metadataHex,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	state := hex.EncodeToString(owner.PubKeyBytes)
	deployScript := artifact.Script + "6a" + state

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
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

	// Attacker tries to burn — should fail checkSig
	spendTx, err := helpers.BuildSpendTx(contractUTXO, attacker.P2PKHScript(), 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	sigHex, err := helpers.SignInput(spendTx, 0, attacker.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}
	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// Unlocking: <opPushTxSig> <sig> <txPreimage> <methodIndex=1>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(1)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	helpers.AssertTxRejected(t, spendTx.Hex())
}
