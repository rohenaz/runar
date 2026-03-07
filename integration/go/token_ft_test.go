//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	runar "github.com/icellan/runar/packages/runar-go"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

// deployFungibleToken compiles and deploys a FungibleToken contract using the SDK,
// returning the contract, provider, signer, and owner wallet. The contract is
// deployed with the given owner and initial balance.
func deployFungibleToken(t *testing.T, owner *helpers.Wallet, initialBalance int64) (*runar.RunarContract, runar.Provider, runar.Signer) {
	t.Helper()

	tokenIdHex := hex.EncodeToString([]byte("TEST-TOKEN-001"))

	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/token-ft/FungibleTokenExample.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("FungibleToken script: %d bytes", len(artifact.Script)/2)

	contract := runar.NewRunarContract(artifact, []interface{}{
		owner.PubKeyHex(),
		int64(initialBalance),
		tokenIdHex,
	})

	wallet := helpers.NewWallet()
	helpers.RPCCall("importaddress", wallet.Address, "", false)
	_, err = helpers.FundWallet(wallet, 1.0)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}

	provider := helpers.NewRPCProvider()
	signer, err := helpers.SDKSignerFromWallet(wallet)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	deployTxid, _, err := contract.Deploy(provider, signer, runar.DeployOptions{Satoshis: 5000})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	t.Logf("deployed: %s", deployTxid)

	return contract, provider, signer
}

// buildFTSpendTx builds a raw spending transaction for a FungibleToken UTXO.
// It creates a continuation output with the new state (new owner + same or new balance).
// Returns the unsigned transaction ready for signing.
func buildFTSpendTx(t *testing.T, contract *runar.RunarContract, newOwnerPubKeyHex string, newBalance int64) *transaction.Transaction {
	t.Helper()

	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("contract has no current UTXO")
	}

	// Build the continuation locking script: code + OP_RETURN + new state
	// The code portion is everything before the last OP_RETURN in the current script.
	lastOpReturn := runar.FindLastOpReturn(utxo.Script)
	if lastOpReturn == -1 {
		t.Fatalf("no OP_RETURN found in contract script")
	}
	codePart := utxo.Script[:lastOpReturn]

	// Serialize the new state: owner (PubKey) + balance (bigint)
	newState := runar.SerializeState(contract.Artifact.StateFields, map[string]interface{}{
		"owner":   newOwnerPubKeyHex,
		"balance": int64(newBalance),
	})
	continuationScript := codePart + "6a" + newState

	lockScript, _ := script.NewFromHex(utxo.Script)
	contScript, _ := script.NewFromHex(continuationScript)
	outputSatoshis := int64(4500)

	spendTx := transaction.NewTransaction()
	spendTx.AddInputWithOutput(&transaction.TransactionInput{
		SourceTXID:       helpers.TxidToChainHash(utxo.Txid),
		SourceTxOutIndex: uint32(utxo.OutputIndex),
		SequenceNumber:   transaction.DefaultSequenceNumber,
	}, &transaction.TransactionOutput{
		Satoshis:      uint64(utxo.Satoshis),
		LockingScript: lockScript,
	})
	spendTx.AddOutput(&transaction.TransactionOutput{
		Satoshis:      uint64(outputSatoshis),
		LockingScript: contScript,
	})

	return spendTx
}

func TestFungibleToken_Compile(t *testing.T) {
	artifact, err := helpers.CompileToSDKArtifact(
		"examples/ts/token-ft/FungibleTokenExample.runar.ts",
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if artifact.ContractName != "FungibleToken" {
		t.Fatalf("expected contract name FungibleToken, got %s", artifact.ContractName)
	}
	t.Logf("FungibleToken compiled: %d bytes", len(artifact.Script)/2)
}

func TestFungibleToken_Deploy(t *testing.T) {
	owner := helpers.NewWallet()
	contract, _, _ := deployFungibleToken(t, owner, 1000)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed with balance=1000")
}

func TestFungibleToken_DeployZeroBalance(t *testing.T) {
	owner := helpers.NewWallet()
	contract, _, _ := deployFungibleToken(t, owner, 0)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed with balance=0")
}

func TestFungibleToken_DeployLargeBalance(t *testing.T) {
	owner := helpers.NewWallet()
	contract, _, _ := deployFungibleToken(t, owner, 99999999999)
	utxo := contract.GetCurrentUtxo()
	if utxo == nil {
		t.Fatalf("no UTXO after deploy")
	}
	t.Logf("deployed with large balance=99999999999")
}

func TestFungibleToken_Send(t *testing.T) {
	// FungibleToken: StatefulSmartContract with addOutput
	// send(sig, to, outputSatoshis) -- transfers entire balance to new owner
	owner := helpers.NewWallet()
	receiver := helpers.NewWallet()
	initialBalance := int64(1000)

	// Deploy using SDK
	contract, _, _ := deployFungibleToken(t, owner, initialBalance)

	// Build raw spend tx (needed because send() takes a checkSig sig argument
	// that must be computed over the final spending transaction)
	spendTx := buildFTSpendTx(t, contract, receiver.PubKeyHex(), initialBalance)
	outputSatoshis := int64(4500)

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

	// Deploy using SDK
	contract, _, _ := deployFungibleToken(t, owner, initialBalance)

	// Build raw spend tx
	spendTx := buildFTSpendTx(t, contract, receiver.PubKeyHex(), initialBalance)
	outputSatoshis := int64(4500)

	// Sign with attacker's key (wrong) -- checkSig should fail
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
		helpers.EncodeMethodIndex(1) // send

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	helpers.AssertTxRejected(t, spendTx.Hex())
}
