"""
SchnorrZKP integration test -- stateless contract with EC scalar math verification.

How It Works
============

SchnorrZKP implements a Schnorr zero-knowledge proof verifier on-chain.
The contract locks funds to an EC public key P, and spending requires proving
knowledge of the discrete logarithm k (i.e., P = k*G) without revealing k.

Constructor
    - pubKey: Point -- the EC public key (64-byte uncompressed x[32] || y[32])

Method: verify(rPoint: Point, s: bigint, e: bigint)
    The prover generates a proof:
        1. Pick random nonce r, compute R = r*G (commitment)
        2. Pick challenge e
        3. Compute s = r + e*k (mod n) (response)
    The contract checks: s*G === R + e*P (Schnorr verification equation)

Script Size
    ~877 KB -- dominated by EC scalar multiplication codegen (each ecMulGen call
    compiles to ~290 KB of Bitcoin Script doing 256 double-and-add iterations).

Important Notes
    - No Sig param -- this is a pure mathematical proof, not an ECDSA signature
    - All params (Point, bigint) are passed as explicit values to contract.call()
    - The contract is stateless (SmartContract base class)
"""

import pytest

from conftest import (
    compile_contract, create_provider, create_funded_wallet,
    ec_mul_gen, encode_point, EC_N,
)
from runar.sdk import RunarContract, DeployOptions


class TestSchnorrZKP:

    def test_compile(self):
        """Compile the SchnorrZKP contract."""
        artifact = compile_contract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts")
        assert artifact
        assert artifact.contract_name == "SchnorrZKP"
        assert len(artifact.script) > 0

    def test_script_size(self):
        """EC-heavy scripts should be approximately 877 KB."""
        artifact = compile_contract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts")
        script_bytes = len(artifact.script) // 2
        assert script_bytes > 100000
        assert script_bytes < 2000000

    def test_deploy(self):
        """Deploy with an EC public key point."""
        artifact = compile_contract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        # Generate keypair: k is private, P = k*G is the public key point
        k = 42
        px, py = ec_mul_gen(k)

        # Constructor: (pubKey: Point) -- 64-byte hex (x[32] || y[32])
        pub_key_hex = encode_point(px, py)
        contract = RunarContract(artifact, [pub_key_hex])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))
        assert txid
        assert isinstance(txid, str)
        assert len(txid) == 64

    def test_deploy_different_key(self):
        """Deploy with a different public key."""
        artifact = compile_contract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        k = 123456789
        px, py = ec_mul_gen(k)
        pub_key_hex = encode_point(px, py)

        contract = RunarContract(artifact, [pub_key_hex])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))
        assert txid

    def test_spend_valid_proof(self):
        """Deploy and spend with a valid Schnorr ZKP proof.

        The proof satisfies the Schnorr verification equation s*G = R + e*P:
            1. Private key k=42, public key P = k*G
            2. Nonce r=7777, commitment R = r*G
            3. Challenge e=12345
            4. Response s = r + e*k mod n
            5. Call verify(R, s, e)
        """
        artifact = compile_contract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        # --- Step 1: Generate keypair ---
        # Private key k (the secret we're proving knowledge of)
        k = 42
        # Public key P = k*G (deployed into the contract)
        px, py = ec_mul_gen(k)
        pub_key_hex = encode_point(px, py)

        contract = RunarContract(artifact, [pub_key_hex])
        contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))

        # --- Step 2: Generate the Schnorr ZKP proof ---
        # Pick a nonce r (deterministic for tests)
        r = 7777
        # Compute R = r*G (nonce commitment)
        rx, ry = ec_mul_gen(r)
        r_point_hex = encode_point(rx, ry)

        # Challenge e (in a real protocol, e = H(R || P || message))
        e = 12345

        # Response s = r + e*k (mod n)
        # This encodes knowledge of k: s*G = r*G + e*k*G = R + e*P
        s = (r + e * k) % EC_N

        # --- Step 3: Call verify(rPoint, s, e) ---
        # The contract checks s*G === R + e*P (always true for valid proof)
        call_txid, _ = contract.call(
            "verify",
            [r_point_hex, s, e],
            provider, wallet["signer"],
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_invalid_s_rejected(self):
        """Verify with tampered s value should be rejected."""
        artifact = compile_contract("examples/ts/schnorr-zkp/SchnorrZKP.runar.ts")

        provider = create_provider()
        wallet = create_funded_wallet(provider)

        k = 42
        px, py = ec_mul_gen(k)
        pub_key_hex = encode_point(px, py)

        contract = RunarContract(artifact, [pub_key_hex])
        contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=50000))

        r = 7777
        rx, ry = ec_mul_gen(r)
        r_point_hex = encode_point(rx, ry)

        e = 12345
        s = (r + e * k) % EC_N

        # Tamper s by adding 1
        tampered_s = (s + 1) % EC_N

        with pytest.raises(Exception):
            contract.call(
                "verify",
                [r_point_hex, tampered_s, e],
                provider, wallet["signer"],
            )
