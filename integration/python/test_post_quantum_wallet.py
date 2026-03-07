"""
PostQuantumWallet integration test -- stateless contract with WOTS+ verification.

How It Works
============

PostQuantumWallet locks funds to a Winternitz One-Time Signature (WOTS+) public key.
WOTS+ is a hash-based post-quantum signature scheme -- its security relies only on
the collision resistance of SHA-256, not on any number-theoretic assumption.

Constructor
    - pubkey: ByteString -- 64-byte hex (pubSeed[32] || pkRoot[32])
        - pubSeed: randomness used in hash chain key derivation
        - pkRoot: SHA-256 hash of all 67 public key chain endpoints

Method: spend(msg: ByteString, sig: ByteString)
    - msg: the message to verify (arbitrary bytes; hashed internally to 32 bytes)
    - sig: 2,144-byte WOTS+ signature (67 chains x 32 bytes each)

How WOTS+ Works
    1. Key gen: 67 random 32-byte secret keys, each chained 15 times (W=16)
    2. Public key = SHA-256(all 67 chain endpoints concatenated)
    3. Sign: hash message to 64 base-16 digits + 3 checksum digits,
       chain each sk[i] forward d[i] steps
    4. Verify: chain each sig[i] the remaining (15 - d[i]) steps,
       hash all endpoints, compare to pkRoot

Script Size
    ~10 KB -- modest because WOTS+ verification is iterative SHA-256 hashing.

Important Notes
    - "One-time": reusing the same keypair for a different message leaks key material
    - No Sig param -- hash-based signature, not ECDSA
"""

import pytest

from conftest import (
    compile_contract, create_provider, create_funded_wallet,
    wots_keygen, wots_sign,
)
from runar.sdk import RunarContract, DeployOptions


class TestPostQuantumWallet:

    def test_compile(self):
        """Compile the PostQuantumWallet contract."""
        artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts")
        assert artifact
        assert artifact.contract_name == "PostQuantumWallet"
        assert len(artifact.script) > 0

    def test_script_size(self):
        """WOTS+ scripts should be approximately 10 KB."""
        artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts")
        script_bytes = len(artifact.script) // 2
        assert script_bytes > 5000
        assert script_bytes < 50000

    def test_deploy(self):
        """Deploy with a WOTS+ public key."""
        artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        # Generate WOTS+ keypair from a deterministic seed
        seed = bytearray(32)
        seed[0] = 0x42
        seed = bytes(seed)
        pub_seed = bytearray(32)
        pub_seed[0] = 0x01
        pub_seed = bytes(pub_seed)

        kp = wots_keygen(seed, pub_seed)

        # Constructor: (pubkey: ByteString) -- 64-byte hex (pubSeed || pkRoot)
        contract = RunarContract(artifact, [kp["pk"]])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=10000))
        assert txid
        assert isinstance(txid, str)
        assert len(txid) == 64

    def test_deploy_different_seed(self):
        """Deploy with a different seed produces a different public key."""
        artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        seed = bytearray(32)
        seed[0] = 0x99
        seed[1] = 0xAB
        seed = bytes(seed)
        pub_seed = bytearray(32)
        pub_seed[0] = 0x02
        pub_seed = bytes(pub_seed)

        kp = wots_keygen(seed, pub_seed)

        contract = RunarContract(artifact, [kp["pk"]])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=10000))
        assert txid

    def test_spend_valid_sig(self):
        """Deploy and spend with a valid WOTS+ signature.

        Steps:
            1. Generate WOTS+ keypair from deterministic seed
            2. Deploy contract with the public key
            3. Sign a message with the WOTS+ secret key (2,144-byte signature)
            4. Call spend(msg, sig) -- the on-chain script verifies by:
               - Hashing the message to 32 bytes (SHA-256)
               - Extracting 64 base-16 digits + 3 checksum digits
               - Chaining each sig[i] forward (15 - d[i]) times
               - Hashing all 67 endpoints, comparing to pkRoot
        """
        artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        # Generate WOTS+ keypair
        seed = bytearray(32)
        seed[0] = 0x42
        seed = bytes(seed)
        pub_seed = bytearray(32)
        pub_seed[0] = 0x01
        pub_seed = bytes(pub_seed)

        kp = wots_keygen(seed, pub_seed)

        # Deploy
        contract = RunarContract(artifact, [kp["pk"]])
        contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=10000))

        # Sign a message with the WOTS+ secret key
        msg = b"spend this UTXO"
        sig = wots_sign(msg, kp["sk"], kp["pubSeed"])

        # WOTS+ signature: 67 chains x 32 bytes = 2,144 bytes
        assert len(sig) == 2144

        # Call spend(msg, sig) to unlock the UTXO
        call_txid, _ = contract.call(
            "spend",
            [msg.hex(), sig.hex()],
            provider, wallet["signer"],
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_tampered_sig_rejected(self):
        """Spend with a tampered WOTS+ signature should be rejected."""
        artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        seed = bytearray(32)
        seed[0] = 0x42
        seed = bytes(seed)
        pub_seed = bytearray(32)
        pub_seed[0] = 0x01
        pub_seed = bytes(pub_seed)

        kp = wots_keygen(seed, pub_seed)

        contract = RunarContract(artifact, [kp["pk"]])
        contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=10000))

        msg = b"spend this UTXO"
        sig = wots_sign(msg, kp["sk"], kp["pubSeed"])

        # Tamper byte 100 (XOR 0xFF)
        tampered = bytearray(sig)
        tampered[100] ^= 0xFF
        tampered = bytes(tampered)

        with pytest.raises(Exception):
            contract.call(
                "spend",
                [msg.hex(), tampered.hex()],
                provider, wallet["signer"],
            )

    def test_wrong_message_rejected(self):
        """Spend with a valid signature but wrong message should be rejected."""
        artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        seed = bytearray(32)
        seed[0] = 0x42
        seed = bytes(seed)
        pub_seed = bytearray(32)
        pub_seed[0] = 0x01
        pub_seed = bytes(pub_seed)

        kp = wots_keygen(seed, pub_seed)

        contract = RunarContract(artifact, [kp["pk"]])
        contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=10000))

        # Sign "original message" but call with "different message"
        original_msg = b"original message"
        sig = wots_sign(original_msg, kp["sk"], kp["pubSeed"])

        different_msg = b"different message"

        with pytest.raises(Exception):
            contract.call(
                "spend",
                [different_msg.hex(), sig.hex()],
                provider, wallet["signer"],
            )
