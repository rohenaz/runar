"""
CovenantVault integration test -- stateless contract with checkSig + checkPreimage.

How It Works
============

CovenantVault demonstrates a covenant pattern: it constrains HOW funds can be spent,
not just WHO can spend them. The contract checks:
    1. The owner's ECDSA signature (authentication via checkSig)
    2. The transaction preimage (via checkPreimage, which enables script-level
       inspection of the spending transaction)
    3. That the spending amount >= minAmount (covenant rule)

What is checkPreimage / OP_PUSH_TX?
    checkPreimage verifies a BIP-143 sighash preimage against the spending transaction.
    This is implemented via the OP_PUSH_TX technique: the unlocking script pushes
    both a preimage (the raw BIP-143 serialization) and an ECDSA signature computed
    with private key k=1 (whose public key is the generator point G). The locking
    script verifies this signature against the preimage, which proves the preimage
    is genuine. Once verified, the script can inspect transaction fields.

Constructor
    - owner: PubKey -- the ECDSA public key that must sign to spend
    - recipient: Addr -- the hash160 of the authorized recipient's public key
    - minAmount: bigint -- minimum satoshis that must be sent to the recipient

Method: spend(sig: Sig, amount: bigint, txPreimage: SigHashPreimage)
    The compiler inserts an implicit _opPushTxSig parameter before the declared params.
    The full unlocking script order is: <opPushTxSig> <sig> <amount> <txPreimage>

    - sig: owner's ECDSA signature (auto-computed by SDK when None)
    - amount: satoshis to send to recipient (must be >= minAmount)
    - txPreimage: BIP-143 sighash preimage (auto-computed by SDK when None)
"""

import pytest

from conftest import (
    compile_contract, create_provider, create_funded_wallet, create_wallet,
)
from runar.sdk import RunarContract, DeployOptions


class TestCovenantVault:

    def test_compile(self):
        """Compile the CovenantVault contract."""
        artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts")
        assert artifact
        assert artifact.contract_name == "CovenantVault"

    def test_deploy(self):
        """Deploy with owner, recipient, and minAmount."""
        artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts")

        provider = create_provider()
        owner = create_wallet()
        recipient = create_wallet()
        wallet = create_funded_wallet(provider)

        # Constructor: (owner: PubKey, recipient: Addr, minAmount: bigint)
        # Addr is a pubKeyHash (20-byte hash160)
        contract = RunarContract(artifact, [
            owner["pubKeyHex"],
            recipient["pubKeyHash"],
            1000,
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid
        assert isinstance(txid, str)
        assert len(txid) == 64

    def test_deploy_zero_min_amount(self):
        """Deploy with zero minAmount."""
        artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts")

        provider = create_provider()
        owner = create_wallet()
        recipient = create_wallet()
        wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            owner["pubKeyHex"],
            recipient["pubKeyHash"],
            0,
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid

    def test_deploy_large_min_amount(self):
        """Deploy with large minAmount (1 BTC in satoshis)."""
        artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts")

        provider = create_provider()
        owner = create_wallet()
        recipient = create_wallet()
        wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            owner["pubKeyHex"],
            recipient["pubKeyHash"],
            100_000_000,
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid

    def test_deploy_same_key_owner_recipient(self):
        """Deploy with the same key as owner and recipient."""
        artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts")

        provider = create_provider()
        both = create_wallet()
        wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            both["pubKeyHex"],
            both["pubKeyHash"],
            500,
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid

    def test_spend_valid(self):
        """Deploy and spend with valid owner signature and amount >= minAmount.

        Steps:
            1. Create owner wallet (will be the signer -- their ECDSA key must match constructor)
            2. Deploy with (ownerPubKey, recipientPubKeyHash, minAmount=1000)
            3. Call spend(None, 2000, None):
               - None Sig -> SDK auto-computes ECDSA signature from signer's private key
               - 2000     -> amount (>= minAmount of 1000)
               - None SigHashPreimage -> SDK auto-computes BIP-143 preimage and _opPushTxSig
            4. The SDK builds the unlocking script: <opPushTxSig> <sig> <amount> <txPreimage>
            5. On-chain, the script verifies checkSig(sig, owner), checkPreimage(txPreimage),
               and asserts amount >= minAmount
        """
        artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts")

        provider = create_provider()
        recipient = create_wallet()

        # Owner must be the signer -- their ECDSA key must match constructor's owner param
        owner_wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            owner_wallet["pubKeyHex"],
            recipient["pubKeyHash"],
            1000,  # minAmount
        ])

        contract.deploy(provider, owner_wallet["signer"], DeployOptions(satoshis=5000))

        # spend(sig=None, amount=2000, txPreimage=None)
        # SDK auto-computes both Sig and SigHashPreimage from the spending transaction
        txid, _ = contract.call(
            "spend",
            [None, 2000, None],
            provider,
            owner_wallet["signer"],
        )
        assert txid
        assert isinstance(txid, str)
        assert len(txid) == 64

    def test_below_min_amount_rejected(self):
        """Spend with amount below minAmount should be rejected."""
        artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts")

        provider = create_provider()
        recipient = create_wallet()
        owner_wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            owner_wallet["pubKeyHex"],
            recipient["pubKeyHash"],
            1000,  # minAmount
        ])

        contract.deploy(provider, owner_wallet["signer"], DeployOptions(satoshis=5000))

        # amount=500 is below minAmount=1000
        with pytest.raises(Exception):
            contract.call(
                "spend",
                [None, 500, None],
                provider,
                owner_wallet["signer"],
            )
