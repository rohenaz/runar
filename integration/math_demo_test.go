//go:build integration

package integration

import (
	"encoding/hex"
	"testing"

	"runar-integration/helpers"

	"github.com/bsv-blockchain/go-sdk/script"
)

func deployMathDemo(t *testing.T, initialValue int64) (*helpers.UTXO, string) {
	t.Helper()
	artifact, err := helpers.CompileContract("examples/ts/math-demo/MathDemo.runar.ts", map[string]interface{}{
		"value": float64(initialValue),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	state := serializeBigintState(initialValue)
	deployScript := artifact.Script + "6a" + state

	funder := helpers.NewWallet()
	funding, err := helpers.FundWallet(funder, 0.01)
	if err != nil {
		t.Fatalf("fund: %v", err)
	}
	deployHex, err := helpers.DeployContract(deployScript, funding, 10000, funder)
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
	return utxo, artifact.Script
}

// spendMathDemo builds a stateful spend for MathDemo with the given new value and method index.
func spendMathDemo(t *testing.T, contractUTXO *helpers.UTXO, newValue int64, methodIdx int, extraArgs string) *helpers.UTXO {
	t.Helper()
	newState := serializeBigintState(newValue)
	continuationScript := buildContinuationScript(contractUTXO.Script, 8, newState)

	spendTx, err := buildStatefulSpendTx(contractUTXO, continuationScript, contractUTXO.Satoshis)
	if err != nil {
		t.Fatalf("build spend: %v", err)
	}

	opPushTxSigHex, preimageHex, err := helpers.SignOpPushTx(spendTx, 0)
	if err != nil {
		t.Fatalf("op_push_tx: %v", err)
	}
	opPushTxSigBytes, _ := hex.DecodeString(opPushTxSigHex)
	preimageBytes, _ := hex.DecodeString(preimageHex)

	// Unlocking: <opPushTxSig> <extraArgs> <txPreimage> <methodIndex>
	// Method params go between opPushTxSig and txPreimage on the stack.
	unlockHex := helpers.EncodePushBytes(opPushTxSigBytes) +
		extraArgs +
		helpers.EncodePushBytes(preimageBytes) +
		helpers.EncodeMethodIndex(methodIdx)

	unlockScript, _ := script.NewFromHex(unlockHex)
	spendTx.Inputs[0].UnlockingScript = unlockScript

	txid, err := helpers.BroadcastAndMine(spendTx.Hex())
	if err != nil {
		t.Fatalf("spend (method %d → value %d): %v", methodIdx, newValue, err)
	}
	t.Logf("value→%d TX: %s", newValue, txid)

	utxo, err := helpers.FindUTXOByIndex(txid, 0)
	if err != nil {
		t.Fatalf("find UTXO: %v", err)
	}
	return utxo
}

func TestMathDemo_DivideBy(t *testing.T) {
	// Deploy value=100, divideBy(5) → value=20
	utxo, _ := deployMathDemo(t, 100)
	t.Logf("MathDemo deployed, script: %d bytes", len(utxo.Script)/2)

	// divideBy is method 0, takes divisor arg
	divisor := int64(5)
	_ = spendMathDemo(t, utxo, 20, 0, helpers.EncodePushInt(divisor))
}

func TestMathDemo_ClampValue(t *testing.T) {
	// Deploy value=500, clampValue(10, 100) → value=100
	utxo, _ := deployMathDemo(t, 500)

	// clampValue is method 2, takes (lo, hi) args
	lo := int64(10)
	hi := int64(100)
	_ = spendMathDemo(t, utxo, 100, 2, helpers.EncodePushInt(lo)+helpers.EncodePushInt(hi))
}

func TestMathDemo_Normalize(t *testing.T) {
	// Deploy value=-42, normalize() → value=-1
	utxo, _ := deployMathDemo(t, -42)

	// normalize is method 3, no extra args
	_ = spendMathDemo(t, utxo, -1, 3, "")
}

func TestMathDemo_SquareRoot(t *testing.T) {
	// Deploy value=144, squareRoot() → value=12
	utxo, _ := deployMathDemo(t, 144)

	// squareRoot is method 5, no extra args
	_ = spendMathDemo(t, utxo, 12, 5, "")
}

func TestMathDemo_Exponentiate(t *testing.T) {
	// Deploy value=2, exponentiate(3) → value=8
	utxo, _ := deployMathDemo(t, 2)

	// exponentiate is method 4, takes exp arg
	exp := int64(3)
	_ = spendMathDemo(t, utxo, 8, 4, helpers.EncodePushInt(exp))
}

func TestMathDemo_ReduceGcd(t *testing.T) {
	// Deploy value=48, reduceGcd(18) → value=6
	utxo, _ := deployMathDemo(t, 48)

	// reduceGcd is method 6, takes other arg
	other := int64(18)
	_ = spendMathDemo(t, utxo, 6, 6, helpers.EncodePushInt(other))
}

func TestMathDemo_ComputeLog2(t *testing.T) {
	// Deploy value=256, computeLog2() → value=8
	utxo, _ := deployMathDemo(t, 256)

	// computeLog2 is method 8, no extra args
	_ = spendMathDemo(t, utxo, 8, 8, "")
}

func TestMathDemo_ScaleByRatio(t *testing.T) {
	// Deploy value=100, scaleByRatio(3, 4) → value=75
	utxo, _ := deployMathDemo(t, 100)

	// scaleByRatio is method 7, takes (numerator, denominator)
	num := int64(3)
	den := int64(4)
	_ = spendMathDemo(t, utxo, 75, 7, helpers.EncodePushInt(num)+helpers.EncodePushInt(den))
}

func TestMathDemo_ChainOperations(t *testing.T) {
	// Deploy value=1000
	// divideBy(10) → 100
	// squareRoot() → 10
	// scaleByRatio(5, 1) → 50
	utxo, _ := deployMathDemo(t, 1000)

	utxo = spendMathDemo(t, utxo, 100, 0, helpers.EncodePushInt(10))
	utxo = spendMathDemo(t, utxo, 10, 5, "")
	_ = spendMathDemo(t, utxo, 50, 7, helpers.EncodePushInt(5)+helpers.EncodePushInt(1))
	t.Logf("chain: 1000→100→10→50 succeeded")
}
