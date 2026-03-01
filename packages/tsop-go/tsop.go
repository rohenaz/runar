// Package tsop provides types and mock functions for TSOP smart contract
// development in Go. Contracts import this package to get IDE support,
// type checking, and the ability to run native Go tests.
//
// Crypto functions like CheckSig are mocked (always return true) to enable
// testing business logic without real signatures. Hash functions (Hash160,
// Hash256, etc.) compute real hashes.
//
// Byte types use string as the underlying type so == comparison works
// naturally in contract code, matching TSOP's === semantics.
package tsop

import (
	"crypto/sha256"

	"golang.org/x/crypto/ripemd160"
)

// ---------------------------------------------------------------------------
// Scalar types — aliases so Go arithmetic operators work directly
// ---------------------------------------------------------------------------

// Int is a TSOP integer (maps to Bitcoin Script numbers).
type Int = int64

// Bigint is an alias for Int.
type Bigint = int64

// Bool is a TSOP boolean.
type Bool = bool

// ---------------------------------------------------------------------------
// Byte-string types — backed by string so == works for equality checks
// ---------------------------------------------------------------------------

// PubKey is a public key (compressed or uncompressed).
type PubKey string

// Sig is a DER-encoded signature.
type Sig string

// Addr is a 20-byte address (typically a hash160 of a public key).
type Addr string

// ByteString is an arbitrary byte sequence.
type ByteString string

// Sha256 is a 32-byte SHA-256 hash.
type Sha256 string

// Ripemd160Hash is a 20-byte RIPEMD-160 hash.
type Ripemd160Hash string

// SigHashPreimage is the sighash preimage for transaction validation.
type SigHashPreimage string

// RabinSig is a Rabin signature.
type RabinSig string

// RabinPubKey is a Rabin public key.
type RabinPubKey string

// ---------------------------------------------------------------------------
// Base contract structs
// ---------------------------------------------------------------------------

// SmartContract is the base struct for stateless TSOP contracts.
// Embed this in your contract struct.
type SmartContract struct{}

// OutputSnapshot records a single output from AddOutput.
type OutputSnapshot struct {
	Satoshis int64
	Values   []any
}

// StatefulSmartContract is the base struct for stateful TSOP contracts.
// Embed this in your contract struct. Provides AddOutput and state tracking.
type StatefulSmartContract struct {
	outputs    []OutputSnapshot
	TxPreimage SigHashPreimage
}

// AddOutput records a new output with the given satoshis and state values.
// The values should match the mutable properties in declaration order.
func (s *StatefulSmartContract) AddOutput(satoshis int64, values ...any) {
	s.outputs = append(s.outputs, OutputSnapshot{
		Satoshis: satoshis,
		Values:   values,
	})
}

// GetStateScript returns a mock state script (empty bytes in test mode).
func (s *StatefulSmartContract) GetStateScript() ByteString {
	return ""
}

// Outputs returns the outputs recorded during the last method execution.
func (s *StatefulSmartContract) Outputs() []OutputSnapshot {
	return s.outputs
}

// ResetOutputs clears recorded outputs (call between test method invocations).
func (s *StatefulSmartContract) ResetOutputs() {
	s.outputs = nil
}

// ---------------------------------------------------------------------------
// Control flow
// ---------------------------------------------------------------------------

// Assert panics if the condition is false, mirroring Bitcoin Script OP_VERIFY.
func Assert(cond bool) {
	if !cond {
		panic("tsop: assertion failed")
	}
}

// ---------------------------------------------------------------------------
// Mock crypto — always succeed for testing business logic
// ---------------------------------------------------------------------------

// CheckSig always returns true in test mode.
func CheckSig(sig Sig, pk PubKey) bool {
	return true
}

// CheckMultiSig always returns true in test mode.
func CheckMultiSig(sigs []Sig, pks []PubKey) bool {
	return true
}

// CheckPreimage always returns true in test mode.
func CheckPreimage(preimage SigHashPreimage) bool {
	return true
}

// VerifyRabinSig always returns true in test mode.
func VerifyRabinSig(msg ByteString, sig RabinSig, padding ByteString, pk RabinPubKey) bool {
	return true
}

// ---------------------------------------------------------------------------
// Real hash functions
// ---------------------------------------------------------------------------

// Hash160 computes RIPEMD160(SHA256(data)), producing a 20-byte address.
func Hash160(data PubKey) Addr {
	h := sha256.Sum256([]byte(data))
	r := ripemd160.New()
	r.Write(h[:])
	return Addr(r.Sum(nil))
}

// Hash256 computes SHA256(SHA256(data)), producing a 32-byte hash.
func Hash256(data ByteString) Sha256 {
	h1 := sha256.Sum256([]byte(data))
	h2 := sha256.Sum256(h1[:])
	return Sha256(h2[:])
}

// Sha256Hash computes a single SHA-256 hash.
func Sha256Hash(data ByteString) Sha256 {
	h := sha256.Sum256([]byte(data))
	return Sha256(h[:])
}

// Ripemd160Func computes a RIPEMD-160 hash.
func Ripemd160Func(data ByteString) Ripemd160Hash {
	r := ripemd160.New()
	r.Write([]byte(data))
	return Ripemd160Hash(r.Sum(nil))
}

// ---------------------------------------------------------------------------
// Mock preimage extraction functions
// ---------------------------------------------------------------------------

// ExtractLocktime returns 0 in test mode.
func ExtractLocktime(p SigHashPreimage) int64 { return 0 }

// ExtractOutputHash returns 32 zero bytes in test mode.
func ExtractOutputHash(p SigHashPreimage) Sha256 { return Sha256(make([]byte, 32)) }

// ExtractAmount returns 10000 in test mode.
func ExtractAmount(p SigHashPreimage) int64 { return 10000 }

// ExtractVersion returns 1 in test mode.
func ExtractVersion(p SigHashPreimage) int64 { return 1 }

// ExtractSequence returns 0xffffffff in test mode.
func ExtractSequence(p SigHashPreimage) int64 { return 0xffffffff }

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

// Num2Bin converts an integer to a byte string of the specified length
// using Bitcoin Script's little-endian signed magnitude encoding.
func Num2Bin(v int64, length int64) ByteString {
	buf := make([]byte, length)
	if v == 0 {
		return ByteString(buf)
	}
	abs := v
	if abs < 0 {
		abs = -abs
	}
	uval := uint64(abs)
	for i := int64(0); i < length && uval > 0; i++ {
		buf[i] = byte(uval & 0xff)
		uval >>= 8
	}
	if v < 0 {
		buf[length-1] |= 0x80
	}
	return ByteString(buf[:length])
}

// Len returns the length of a byte string as an integer.
func Len(data ByteString) int64 {
	return int64(len(data))
}

// Cat concatenates two byte strings.
func Cat(a, b ByteString) ByteString {
	return a + b
}

// Substr returns a substring of a byte string.
func Substr(data ByteString, start, length int64) ByteString {
	return data[start : start+length]
}

// ReverseBytes returns a reversed copy of a byte string.
func ReverseBytes(data ByteString) ByteString {
	b := []byte(data)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return ByteString(b)
}

// Abs returns the absolute value.
func Abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// Min returns the smaller of two values.
func Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of two values.
func Max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Within returns true if min <= value < max.
func Within(value, min, max int64) bool {
	return value >= min && value < max
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// MockSig returns a dummy signature for testing.
func MockSig() Sig {
	return Sig(make([]byte, 72))
}

// MockPubKey returns a dummy compressed public key for testing.
func MockPubKey() PubKey {
	pk := make([]byte, 33)
	pk[0] = 0x02
	return PubKey(pk)
}

// MockPreimage returns a dummy sighash preimage for testing.
func MockPreimage() SigHashPreimage {
	return SigHashPreimage(make([]byte, 181))
}
