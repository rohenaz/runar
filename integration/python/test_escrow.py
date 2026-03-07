"""
Escrow integration test -- stateless contract with checkSig.

Escrow locks funds and allows release or refund via four methods, each
requiring a signature from the appropriate party. The SDK auto-computes
Sig params when None is passed.
"""

import pytest

from conftest import (
    compile_contract, create_provider, create_funded_wallet, create_wallet,
)
from runar.sdk import RunarContract, DeployOptions


class TestEscrow:

    def test_compile(self):
        """Compile the Escrow contract."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")
        assert artifact
        assert artifact.contract_name == "Escrow"

    def test_deploy_three_pubkeys(self):
        """Deploy with three distinct pubkeys (buyer, seller, arbiter)."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")

        provider = create_provider()
        buyer = create_wallet()
        seller = create_wallet()
        arbiter = create_wallet()
        wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            buyer["pubKeyHex"],
            seller["pubKeyHex"],
            arbiter["pubKeyHex"],
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid
        assert isinstance(txid, str)
        assert len(txid) == 64

    def test_deploy_same_key_multiple_roles(self):
        """Deploy with the same key as both buyer and arbiter."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")

        provider = create_provider()
        buyer_and_arbiter = create_wallet()
        seller = create_wallet()
        wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            buyer_and_arbiter["pubKeyHex"],
            seller["pubKeyHex"],
            buyer_and_arbiter["pubKeyHex"],
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid

    def test_release_by_seller(self):
        """Deploy and spend via releaseBySeller with auto-computed Sig."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")

        provider = create_provider()
        buyer = create_wallet()
        arbiter = create_wallet()
        # Seller is the funded signer, so the auto-computed sig matches
        seller_wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            buyer["pubKeyHex"],
            seller_wallet["pubKeyHex"],
            arbiter["pubKeyHex"],
        ])

        contract.deploy(provider, seller_wallet["signer"], DeployOptions(satoshis=5000))

        # None Sig is auto-computed by the SDK from the signer (who is the seller)
        call_txid, _ = contract.call(
            "releaseBySeller", [None], provider, seller_wallet["signer"],
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_release_by_arbiter(self):
        """Deploy and spend via releaseByArbiter with auto-computed Sig."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")

        provider = create_provider()
        buyer = create_wallet()
        seller = create_wallet()
        arbiter_wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            buyer["pubKeyHex"],
            seller["pubKeyHex"],
            arbiter_wallet["pubKeyHex"],
        ])

        contract.deploy(provider, arbiter_wallet["signer"], DeployOptions(satoshis=5000))

        call_txid, _ = contract.call(
            "releaseByArbiter", [None], provider, arbiter_wallet["signer"],
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_refund_to_buyer(self):
        """Deploy and spend via refundToBuyer with auto-computed Sig."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")

        provider = create_provider()
        seller = create_wallet()
        arbiter = create_wallet()
        buyer_wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            buyer_wallet["pubKeyHex"],
            seller["pubKeyHex"],
            arbiter["pubKeyHex"],
        ])

        contract.deploy(provider, buyer_wallet["signer"], DeployOptions(satoshis=5000))

        call_txid, _ = contract.call(
            "refundToBuyer", [None], provider, buyer_wallet["signer"],
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_refund_by_arbiter(self):
        """Deploy and spend via refundByArbiter with auto-computed Sig."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")

        provider = create_provider()
        buyer = create_wallet()
        seller = create_wallet()
        arbiter_wallet = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            buyer["pubKeyHex"],
            seller["pubKeyHex"],
            arbiter_wallet["pubKeyHex"],
        ])

        contract.deploy(provider, arbiter_wallet["signer"], DeployOptions(satoshis=5000))

        call_txid, _ = contract.call(
            "refundByArbiter", [None], provider, arbiter_wallet["signer"],
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_wrong_signer_rejected(self):
        """releaseBySeller with wrong signer should be rejected."""
        artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts")

        provider = create_provider()
        seller = create_wallet()
        arbiter = create_wallet()
        wallet_a = create_funded_wallet(provider)
        wallet_b = create_funded_wallet(provider)

        contract = RunarContract(artifact, [
            wallet_a["pubKeyHex"],
            wallet_a["pubKeyHex"],
            arbiter["pubKeyHex"],
        ])

        contract.deploy(provider, wallet_a["signer"], DeployOptions(satoshis=5000))

        with pytest.raises(Exception):
            contract.call(
                "releaseBySeller", [None], provider, wallet_b["signer"],
            )
