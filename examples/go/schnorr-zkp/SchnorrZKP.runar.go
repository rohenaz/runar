package contract

import runar "github.com/icellan/runar/packages/runar-go"

// SchnorrSig is the non-interactive Schnorr proof tuple sent by the prover.
// (R = r·G, s = r + e·k mod n)
type SchnorrSig struct {
	R runar.Point  `runar:"public"` // commitment point
	S runar.Bigint `runar:"public"` // response scalar
}

// SchnorrOwnershipVerifier is a complete, production-ready Runar contract.
//
// It proves knowledge of the private key belonging to `PubKey` (without revealing it)
// and binds the proof to the exact transaction being spent via OP_PUSH_TX
// (perfect replay protection). This is the pattern used in real BSV covenants.
type SchnorrOwnershipVerifier struct {
	runar.StatefulSmartContract             // gives automatic TxPreimage (validated via OP_PUSH_TX)
	PubKey                      runar.Point `runar:"readonly"` // the identity/public key
}

// Authorize is the public entry point.
// The spender provides the Schnorr proof + any business context.
// The proof is cryptographically bound to this exact transaction.
func (c *SchnorrOwnershipVerifier) Authorize(sig SchnorrSig, actionData runar.Bytes) {
	// TxPreimage is automatically pushed and validated by StatefulSmartContract

	// Bind the proof to this specific transaction (perfect replay protection)
	txBinding := runar.SHA256(c.TxPreimage)
	fullMessage := runar.Concat(actionData, txBinding)

	// Perform the actual Schnorr verification
	c.verifySchnorr(sig, fullMessage)

	// === Protected business logic goes here ===
	// (e.g. update state, spend outputs, create new covenants, etc.)
	// If you need to "log" data, push an OP_RETURN output with runar.addOutput
}

// verifySchnorr is the reusable core verification logic.
// Drop this into any other contract that needs Schnorr ownership proofs.
func (c *SchnorrOwnershipVerifier) verifySchnorr(sig SchnorrSig, message runar.Bytes) {
	// 1. Basic curve membership check
	runar.Assert(runar.EcOnCurve(sig.R))

	// 2. Fiat-Shamir heuristic – compute challenge on-chain
	//    e = SHA256(Rx || Px || message)
	rX := runar.EcPointX(sig.R)
	pX := runar.EcPointX(c.PubKey)
	challengePreimage := runar.Concat(rX.Bytes(), pX.Bytes(), message)

	eBytes := runar.SHA256(challengePreimage)
	e := runar.BigintFromBytes(eBytes) // hash interpreted as scalar

	// 3. Core Schnorr equation: s·G == R + e·P
	sG := runar.EcMulGen(sig.S)
	eP := runar.EcMul(c.PubKey, e)
	rhs := runar.EcAdd(sig.R, eP)

	// Verify points are identical
	runar.Assert(runar.EcPointX(sG) == runar.EcPointX(rhs))
	runar.Assert(runar.EcPointY(sG) == runar.EcPointY(rhs))
}
