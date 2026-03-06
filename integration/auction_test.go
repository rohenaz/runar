//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"
)

func compileAuction(t *testing.T, auctioneer *helpers.Wallet, highestBidder *helpers.Wallet, highestBid, deadline int64) (string, string) {
	t.Helper()
	artifact, err := helpers.CompileContract("examples/ts/auction/Auction.runar.ts", map[string]interface{}{
		"auctioneer":    auctioneer.PubKeyHex(),
		"highestBidder": highestBidder.PubKeyHex(),
		"highestBid":    float64(highestBid),
		"deadline":      float64(deadline),
	})
	if err != nil {
		t.Fatalf("compile auction: %v", err)
	}
	return artifact.Script, artifact.Script
}

func deployAuction(t *testing.T, scriptHex string, highestBid, deadline int64, funder *helpers.Wallet) *helpers.UTXO {
	t.Helper()
	// Stateful: code + OP_RETURN + state
	// State: auctioneer(33 bytes PubKey) + highestBidder(33 bytes PubKey) + highestBid(8 bytes) + deadline(8 bytes)
	// But the readonly props (auctioneer, deadline) are in the code part via constructor slots.
	// Only mutable props (highestBidder, highestBid) are in the state section.
	// State = highestBidder(33 bytes) + highestBid(8 bytes)
	state := hex.EncodeToString(funder.PubKeyBytes) + serializeBigintState(highestBid)
	deployScript := scriptHex + "6a" + state

	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
	deployHex, err := helpers.DeployContract(deployScript, funding, 5000, funder)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	txid, err := helpers.BroadcastAndMine(deployHex)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	utxo, err := helpers.FindUTXOByIndex(txid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}
	return utxo
}

func TestAuction_Close(t *testing.T) {
	// Deploy auction, then close it with the auctioneer's signature.
	// Uses method index 1 (close).
	auctioneer := helpers.NewWallet()
	bidder := helpers.NewWallet()

	artifact, err := helpers.CompileContract("examples/ts/auction/Auction.runar.ts", map[string]interface{}{
		"auctioneer":    auctioneer.PubKeyHex(),
		"highestBidder": bidder.PubKeyHex(),
		"highestBid":    float64(1000),
		"deadline":      float64(0),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("Auction script: %d bytes", len(artifact.Script)/2)

	// Deploy with mutable state: highestBidder + highestBid
	state := hex.EncodeToString(bidder.PubKeyBytes) + serializeBigintState(1000)
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

	// Build spend TX: close pays out to auctioneer
	receiverScript := auctioneer.P2PKHScript()
	spendTx, err := helpers.BuildSpendTx(contractUTXO, receiverScript, 4500)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	sigHex, err := helpers.SignInput(spendTx, 0, auctioneer.PrivKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigBytes, _ := hex.DecodeString(sigHex)

	// close(sig: Sig) — method index 1
	// OP_PUSH_TX for stateful
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
		helpers.EncodeMethodIndex(1) // close

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	txid := helpers.AssertTxAccepted(t, spendTx.Hex())
	helpers.AssertTxInBlock(t, txid)
}

func TestAuction_WrongSigner_Rejected(t *testing.T) {
	auctioneer := helpers.NewWallet()
	bidder := helpers.NewWallet()
	attacker := helpers.NewWallet()

	artifact, err := helpers.CompileContract("examples/ts/auction/Auction.runar.ts", map[string]interface{}{
		"auctioneer":    auctioneer.PubKeyHex(),
		"highestBidder": bidder.PubKeyHex(),
		"highestBid":    float64(1000),
		"deadline":      float64(0),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	state := hex.EncodeToString(bidder.PubKeyBytes) + serializeBigintState(1000)
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

	// Attacker tries to close — should fail checkSig
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
		helpers.EncodeMethodIndex(1) // close

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	helpers.AssertTxRejected(t, spendTx.Hex())
}
