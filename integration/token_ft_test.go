//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

func TestFungibleToken_Send(t *testing.T) {
	// FungibleToken: StatefulSmartContract with addOutput
	// send(sig, to, outputSatoshis) — transfers entire balance to new owner
	owner := helpers.NewWallet()
	receiver := helpers.NewWallet()
	initialBalance := int64(1000)
	tokenIdHex := hex.EncodeToString([]byte("TEST-TOKEN-001"))

	artifact, err := helpers.CompileContract("examples/ts/token-ft/FungibleTokenExample.runar.ts", map[string]interface{}{
		"owner":   owner.PubKeyHex(),
		"balance": float64(initialBalance),
		"tokenId": tokenIdHex,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("FungibleToken script: %d bytes", len(artifact.Script)/2)

	// State = owner(33 bytes PubKey) + balance(8 bytes LE via NUM2BIN)
	// Mutable fields in declaration order: owner, balance
	// tokenId is readonly (baked into code via constructor)
	ownerState := hex.EncodeToString(owner.PubKeyBytes)
	balanceState := serializeBigintState(initialBalance)
	state := ownerState + balanceState
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

	// send(sig, to, outputSatoshis) — creates one output with new owner
	// The continuation output scriptPubKey = codePart + OP_RETURN + newState
	newOwnerState := hex.EncodeToString(receiver.PubKeyBytes)
	newState := newOwnerState + balanceState // same balance, new owner
	continuationScript := buildContinuationScript(contractUTXO.Script, len(state)/2, newState)

	lockScript, _ := script.NewFromHex(contractUTXO.Script)
	contScript, _ := script.NewFromHex(continuationScript)
	outputSatoshis := int64(4500)

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

	// send is method index 1 (transfer=0, send=1, merge=2)
	// Unlocking: <opPushTxSig> <sig> <to:PubKey> <outputSatoshis> <txPreimage> <methodIdx>
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushBytes(receiver.PubKeyBytes) +
		helpers.EncodePushInt(outputSatoshis) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(1) // send

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	txid := helpers.AssertTxAccepted(t, spendTx.Hex())
	helpers.AssertTxInBlock(t, txid)
}

func TestFungibleToken_WrongOwner_Rejected(t *testing.T) {
	owner := helpers.NewWallet()
	attacker := helpers.NewWallet()
	receiver := helpers.NewWallet()
	initialBalance := int64(1000)
	tokenIdHex := hex.EncodeToString([]byte("TEST-TOKEN-002"))

	artifact, err := helpers.CompileContract("examples/ts/token-ft/FungibleTokenExample.runar.ts", map[string]interface{}{
		"owner":   owner.PubKeyHex(),
		"balance": float64(initialBalance),
		"tokenId": tokenIdHex,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ownerState := hex.EncodeToString(owner.PubKeyBytes)
	balanceState := serializeBigintState(initialBalance)
	state := ownerState + balanceState
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

	// Attacker tries to send — checkSig should fail
	newOwnerState := hex.EncodeToString(receiver.PubKeyBytes)
	newState := newOwnerState + balanceState
	continuationScript := buildContinuationScript(contractUTXO.Script, len(state)/2, newState)

	lockScript, _ := script.NewFromHex(contractUTXO.Script)
	contScript, _ := script.NewFromHex(continuationScript)
	outputSatoshis := int64(4500)

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

	// Sign with attacker's key (wrong)
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

	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		helpers.EncodePushBytes(sigBytes) +
		helpers.EncodePushBytes(receiver.PubKeyBytes) +
		helpers.EncodePushInt(outputSatoshis) +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(1)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	helpers.AssertTxRejected(t, spendTx.Hex())
}
