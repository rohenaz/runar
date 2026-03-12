package runar

import (
	"bytes"
	"testing"
)

func TestWotsKeygen_KeyLengths(t *testing.T) {
	seed := make([]byte, 32)
	pubSeed := make([]byte, 32)
	kp := WotsKeygen(seed, pubSeed)

	if len(kp.SK) != wotsLen {
		t.Fatalf("expected %d secret key elements, got %d", wotsLen, len(kp.SK))
	}
	for i, sk := range kp.SK {
		if len(sk) != wotsN {
			t.Fatalf("secret key element %d: expected %d bytes, got %d", i, wotsN, len(sk))
		}
	}
	if len(kp.PK) != 2*wotsN {
		t.Fatalf("expected public key %d bytes, got %d", 2*wotsN, len(kp.PK))
	}
	if len(kp.PubSeed) != wotsN {
		t.Fatalf("expected pubSeed %d bytes, got %d", wotsN, len(kp.PubSeed))
	}
}

func TestWotsKeygen_PubSeedInPK(t *testing.T) {
	seed := make([]byte, 32)
	pubSeed := make([]byte, 32)
	for i := range pubSeed {
		pubSeed[i] = byte(i)
	}
	kp := WotsKeygen(seed, pubSeed)

	if !bytes.Equal(kp.PK[:wotsN], pubSeed) {
		t.Fatal("first 32 bytes of PK should equal pubSeed")
	}
}

func TestWotsKeygen_DeterministicWithSeed(t *testing.T) {
	seed := []byte("deterministic-seed-for-wots-test!")
	pubSeed := make([]byte, 32)
	kp1 := WotsKeygen(seed, pubSeed)
	kp2 := WotsKeygen(seed, pubSeed)

	if !bytes.Equal(kp1.PK, kp2.PK) {
		t.Fatal("same seed should produce same public key")
	}
	for i := range kp1.SK {
		if !bytes.Equal(kp1.SK[i], kp2.SK[i]) {
			t.Fatalf("same seed should produce same secret key element %d", i)
		}
	}
}

func TestWotsSignVerify_Valid(t *testing.T) {
	seed := []byte("test-seed-for-wots-sign-verify!!")
	pubSeed := make([]byte, 32)
	kp := WotsKeygen(seed, pubSeed)

	msg := []byte("hello, WOTS+ verification test!")
	sig := WotsSign(msg, kp.SK, kp.PubSeed)

	if len(sig) != wotsLen*wotsN {
		t.Fatalf("expected signature %d bytes, got %d", wotsLen*wotsN, len(sig))
	}

	if !wotsVerifyImpl(msg, sig, kp.PK) {
		t.Fatal("valid WOTS+ signature should verify")
	}
}

func TestVerifyWOTS_ValidViaPublicAPI(t *testing.T) {
	seed := []byte("test-seed-for-public-api-verify!")
	pubSeed := make([]byte, 32)
	kp := WotsKeygen(seed, pubSeed)

	msg := []byte("testing VerifyWOTS public function")
	sig := WotsSign(msg, kp.SK, kp.PubSeed)

	if !VerifyWOTS(ByteString(msg), ByteString(sig), ByteString(kp.PK)) {
		t.Fatal("VerifyWOTS should return true for valid signature")
	}
}

func TestWotsVerify_WrongMessage(t *testing.T) {
	seed := []byte("test-seed-for-wrong-message-chk!")
	pubSeed := make([]byte, 32)
	kp := WotsKeygen(seed, pubSeed)

	msg := []byte("original message for signing test")
	sig := WotsSign(msg, kp.SK, kp.PubSeed)

	wrongMsg := []byte("different message than was signed")
	if wotsVerifyImpl(wrongMsg, sig, kp.PK) {
		t.Fatal("WOTS+ verification should fail for wrong message")
	}
}

func TestWotsVerify_TamperedSignature(t *testing.T) {
	seed := []byte("test-seed-for-tampered-sig-test!")
	pubSeed := make([]byte, 32)
	kp := WotsKeygen(seed, pubSeed)

	msg := []byte("message for tampered sig testing")
	sig := WotsSign(msg, kp.SK, kp.PubSeed)

	// Flip a byte in the signature
	tampered := make([]byte, len(sig))
	copy(tampered, sig)
	tampered[0] ^= 0xff

	if wotsVerifyImpl(msg, tampered, kp.PK) {
		t.Fatal("WOTS+ verification should fail for tampered signature")
	}
}

func TestWotsVerify_BadSignatureLength(t *testing.T) {
	pubSeed := make([]byte, 32)
	pk := make([]byte, 2*wotsN)
	copy(pk, pubSeed)

	if wotsVerifyImpl([]byte("msg"), []byte("short"), pk) {
		t.Fatal("should reject signature with wrong length")
	}
}

func TestWotsVerify_BadPublicKeyLength(t *testing.T) {
	sig := make([]byte, wotsLen*wotsN)
	if wotsVerifyImpl([]byte("msg"), sig, []byte("short-pk")) {
		t.Fatal("should reject public key with wrong length")
	}
}

func TestWotsExtractDigits_Length(t *testing.T) {
	hash := make([]byte, 32)
	digits := wotsExtractDigits(hash)
	if len(digits) != wotsLen1 {
		t.Fatalf("expected %d digits, got %d", wotsLen1, len(digits))
	}
}

func TestWotsExtractDigits_Range(t *testing.T) {
	// Fill hash with 0xff to get max digit values
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = 0xff
	}
	digits := wotsExtractDigits(hash)
	for i, d := range digits {
		if d < 0 || d >= wotsW {
			t.Fatalf("digit %d out of range [0, %d): got %d", i, wotsW, d)
		}
	}
}

func TestWotsChecksumDigits_Length(t *testing.T) {
	msgDigits := make([]int, wotsLen1)
	csum := wotsChecksumDigits(msgDigits)
	if len(csum) != wotsLen2 {
		t.Fatalf("expected %d checksum digits, got %d", wotsLen2, len(csum))
	}
}

func TestWotsAllDigits_TotalLength(t *testing.T) {
	hash := make([]byte, 32)
	all := wotsAllDigits(hash)
	if len(all) != wotsLen {
		t.Fatalf("expected %d total digits, got %d", wotsLen, len(all))
	}
}
