"""
FungibleToken integration test -- stateful contract with addOutput.

FungibleToken is a StatefulSmartContract with properties:
    - owner: PubKey (mutable)
    - balance: bigint (mutable)
    - tokenId: ByteString (readonly)

The SDK auto-computes Sig params when None is passed.
"""

import pytest

from conftest import (
    compile_contract, create_provider, create_funded_wallet, create_wallet,
)
from runar.sdk import RunarContract, DeployOptions, CallOptions


class TestFungibleToken:

    def test_compile(self):
        """Compile the FungibleToken contract."""
        artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts")
        assert artifact
        assert artifact.contract_name == "FungibleToken"

    def test_deploy_with_owner_and_balance(self):
        """Deploy with owner and initial balance of 1000."""
        artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts")

        provider = create_provider()
        owner = create_wallet()
        wallet = create_funded_wallet(provider)

        token_id_hex = b"TEST-TOKEN-001".hex()

        contract = RunarContract(artifact, [
            owner["pubKeyHex"],
            1000,
            token_id_hex,
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid
        assert isinstance(txid, str)
        assert len(txid) == 64

    def test_deploy_zero_balance(self):
        """Deploy with zero initial balance."""
        artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts")

        provider = create_provider()
        owner = create_wallet()
        wallet = create_funded_wallet(provider)

        token_id_hex = b"ZERO-BAL-TOKEN".hex()

        contract = RunarContract(artifact, [
            owner["pubKeyHex"],
            0,
            token_id_hex,
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid

    def test_deploy_large_balance(self):
        """Deploy with a very large balance (21M BTC in satoshis)."""
        artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts")

        provider = create_provider()
        owner = create_wallet()
        wallet = create_funded_wallet(provider)

        token_id_hex = b"BIG-TOKEN".hex()

        contract = RunarContract(artifact, [
            owner["pubKeyHex"],
            21000000_00000000,
            token_id_hex,
        ])

        txid, _ = contract.deploy(provider, wallet["signer"], DeployOptions(satoshis=5000))
        assert txid

    def test_send(self):
        """Deploy and send entire balance to a recipient."""
        artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts")

        provider = create_provider()
        owner_wallet = create_funded_wallet(provider)
        recipient = create_wallet()

        token_id_hex = b"SEND-TOKEN".hex()

        # Owner is the funded signer
        contract = RunarContract(artifact, [
            owner_wallet["pubKeyHex"],
            1000,
            token_id_hex,
        ])

        contract.deploy(provider, owner_wallet["signer"], DeployOptions(satoshis=5000))

        # send: sig=None (auto), to=recipient, outputSatoshis=5000
        # send uses addOutput: on-chain script expects output with owner=to, balance=1000
        # Pass new_state so the SDK builds the correct continuation output
        call_txid, _ = contract.call(
            "send",
            [None, recipient["pubKeyHex"], 5000],
            provider, owner_wallet["signer"],
            options=CallOptions(new_state={"owner": recipient["pubKeyHex"]}),
        )
        assert call_txid
        assert len(call_txid) == 64

    def test_wrong_owner_rejected(self):
        """Send with wrong signer (not the owner) should be rejected."""
        artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts")

        provider = create_provider()
        owner_wallet = create_funded_wallet(provider)
        wrong_wallet = create_funded_wallet(provider)
        recipient = create_wallet()

        token_id_hex = b"REJECT-TOKEN".hex()

        contract = RunarContract(artifact, [
            owner_wallet["pubKeyHex"],
            1000,
            token_id_hex,
        ])

        contract.deploy(provider, owner_wallet["signer"], DeployOptions(satoshis=5000))

        with pytest.raises(Exception):
            contract.call(
                "send",
                [None, recipient["pubKeyHex"], 5000],
                provider, wrong_wallet["signer"],
                options=CallOptions(new_state={"owner": recipient["pubKeyHex"]}),
            )
