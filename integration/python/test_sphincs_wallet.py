"""
SPHINCSWallet integration test -- stateless contract with SLH-DSA-SHA2-128s verification.

How It Works
============

SPHINCSWallet locks funds to an SLH-DSA public key (FIPS 205, 128-bit post-quantum
security level). Unlike WOTS+ (one-time), the same SLH-DSA keypair can sign many
messages because it uses a Merkle tree of WOTS+ keys internally.

Constructor
    - pubkey: ByteString -- 32-byte hex (PK.seed[16] || PK.root[16])

Method: spend(msg: ByteString, sig: ByteString)
    - msg: the signed message (arbitrary bytes)
    - sig: 7,856-byte SLH-DSA-SHA2-128s signature
    The contract verifies the SLH-DSA signature on-chain using ~188 KB of Bitcoin Script.

Script Size
    ~188 KB -- SLH-DSA verification requires computing multiple WOTS+ verifications
    and Merkle tree path checks within the Bitcoin Script VM.

Test Approach
    Uses a pre-computed test vector from conformance/testdata/slhdsa-test-sig.hex
    with a known public key and message, avoiding the need for a full SLH-DSA
    signing library. The same test vector is used by the Go integration tests.
"""

from pathlib import Path

import pytest

from conftest import compile_contract, create_provider, create_funded_wallet
from runar.sdk import RunarContract, DeployOptions


# Deterministic test public key (32 bytes hex: PK.seed || PK.root)
SLHDSA_TEST_PK = "00000000000000000000000000000000b618cb38f7f785488c9768f3a2972baf"
# Message that was signed: "slh-dsa test vector" in hex
SLHDSA_TEST_MSG = "736c682d647361207465737420766563746f72"


def load_test_signature() -> str:
    """Load the pre-computed SLH-DSA test signature from conformance test data.

    Returns the hex string (15,712 chars = 7,856 bytes).
    """
    sig_path = Path(__file__).resolve().parent.parent.parent / "conformance" / "testdata" / "slhdsa-test-sig.hex"
    return sig_path.read_text().strip()


class TestSPHINCSWallet:

    def test_compile(self):
        """Compile the SPHINCSWallet contract."""
        artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts")
        assert artifact
        assert artifact.contract_name == "SPHINCSWallet"
        assert len(artifact.script) > 0

    def test_script_size(self):
        """SLH-DSA scripts should be approximately 188 KB."""
        artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts")
        script_bytes = len(artifact.script) // 2
        assert script_bytes > 100000
        assert script_bytes < 500000

    def test_deploy(self):
        """Deploy with an SLH-DSA public key."""
        artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        # Constructor: (pubkey: ByteString) -- 32-byte hex
        contract = RunarContract(artifact, [SLHDSA_TEST_PK])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))
        assert txid
        assert isinstance(txid, str)
        assert len(txid) == 64

    def test_deploy_different_key(self):
        """Deploy with a different public key."""
        artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        other_pk = "aabbccdd00000000000000000000000011223344556677889900aabbccddeeff"
        contract = RunarContract(artifact, [other_pk])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))
        assert txid

    def test_spend_valid_sig(self):
        """Deploy and spend with a valid SLH-DSA signature (pre-computed test vector).

        The signature was generated offline with the matching private key.
        SLH-DSA-SHA2-128s signatures are 7,856 bytes (FIPS 205 Table 2).

        The on-chain script verifies by:
            1. Parsing the sig into FORS trees + Hypertree layers
            2. Computing WOTS+ public keys from signature chains
            3. Verifying Merkle tree authentication paths
            4. Comparing the reconstructed root against PK.root
        """
        artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        # Deploy with the known test public key
        contract = RunarContract(artifact, [SLHDSA_TEST_PK])
        contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))

        # Load the pre-computed test signature
        sig_hex = load_test_signature()
        assert len(sig_hex) // 2 == 7856, f"SLH-DSA sig must be 7,856 bytes, got {len(sig_hex) // 2}"

        # Call spend(msg, sig) to unlock the UTXO
        call_txid, _ = contract.call(
            "spend",
            [SLHDSA_TEST_MSG, sig_hex],
            provider, wallet["signer"],
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_tampered_sig_rejected(self):
        """Spend with a tampered SLH-DSA signature should be rejected."""
        artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [SLHDSA_TEST_PK])
        contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))

        sig_hex = load_test_signature()

        # Tamper byte 500 (XOR 0xFF)
        sig_bytes = bytearray(bytes.fromhex(sig_hex))
        sig_bytes[500] ^= 0xFF
        tampered_hex = bytes(sig_bytes).hex()

        with pytest.raises(Exception):
            contract.call(
                "spend",
                [SLHDSA_TEST_MSG, tampered_hex],
                provider, wallet["signer"],
            )
