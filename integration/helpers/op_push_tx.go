package helpers

import (
	"encoding/hex"
	"fmt"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sighash "github.com/bsv-blockchain/go-sdk/transaction/sighash"
)

var (
	opPushTxKey    *ec.PrivateKey
	opPushTxPubKey *ec.PublicKey
)

func init() {
	// OP_PUSH_TX uses private key = 1 (public key = generator point G).
	keyBytes := make([]byte, 32)
	keyBytes[31] = 1
	opPushTxKey, opPushTxPubKey = ec.PrivateKeyFromBytes(keyBytes)
}

// SignOpPushTx signs the BIP-143 sighash with the OP_PUSH_TX key (k=1).
// Returns (sigWithFlagHex, preimageHex, error).
func SignOpPushTx(tx *transaction.Transaction, inputIdx uint32) (string, string, error) {
	// Get the raw preimage bytes (pushed onto the stack for checkPreimage)
	preimage, err := tx.CalcInputPreimage(inputIdx, sighash.AllForkID)
	if err != nil {
		return "", "", fmt.Errorf("calc preimage: %w", err)
	}

	// Get the sighash (double-SHA256 of preimage) for signing
	sigHash, err := tx.CalcInputSignatureHash(inputIdx, sighash.AllForkID)
	if err != nil {
		return "", "", fmt.Errorf("calc sighash: %w", err)
	}

	sig, err := opPushTxKey.Sign(sigHash)
	if err != nil {
		return "", "", fmt.Errorf("sign: %w", err)
	}

	sigBytes := sig.Serialize()
	sigBytes = append(sigBytes, byte(sighash.AllForkID))

	return hex.EncodeToString(sigBytes), hex.EncodeToString(preimage), nil
}

// OpPushTxPubKeyHex returns the compressed public key hex for OP_PUSH_TX (generator point G).
func OpPushTxPubKeyHex() string {
	return hex.EncodeToString(opPushTxPubKey.Compressed())
}
