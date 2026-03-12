"""Tests for runar.sdk.calling — transaction construction for method invocation."""

import pytest
from runar.sdk.calling import build_call_transaction, insert_unlocking_script
from runar.sdk.types import Utxo


def _make_utxo(satoshis: int, index: int = 0) -> Utxo:
    txid = f'{index:02x}' * 32
    return Utxo(txid=txid, output_index=0, satoshis=satoshis, script='76a914' + '00' * 20 + '88ac')


# ---------------------------------------------------------------------------
# build_call_transaction
# ---------------------------------------------------------------------------

class TestBuildCallTransaction:
    def test_basic_call_returns_valid_hex(self):
        """A basic call transaction returns (tx_hex, input_count, change_amount)."""
        utxo = _make_utxo(50_000)
        tx_hex, input_count, change_amount = build_call_transaction(
            current_utxo=utxo,
            unlocking_script='5151',  # OP_TRUE OP_TRUE
            new_locking_script='51',
            new_satoshis=10_000,
            change_address='00' * 20,
        )

        assert isinstance(tx_hex, str)
        assert len(tx_hex) > 0
        assert all(c in '0123456789abcdef' for c in tx_hex)
        assert input_count == 1
        assert isinstance(change_amount, int)
        # Starts with version 01000000
        assert tx_hex[:8] == '01000000'

    def test_with_additional_utxos(self):
        """Additional funding UTXOs appear as extra inputs."""
        utxo = _make_utxo(10_000, 0)
        funding = [_make_utxo(50_000, 1)]

        tx_hex, input_count, change_amount = build_call_transaction(
            current_utxo=utxo,
            unlocking_script='51',
            new_locking_script='51',
            new_satoshis=10_000,
            change_address='00' * 20,
            additional_utxos=funding,
        )

        assert input_count == 2
        # After version (8 hex), varint should be 02
        assert tx_hex[8:10] == '02'

    def test_with_contract_outputs(self):
        """Multi-output calls pass contract_outputs list."""
        utxo = _make_utxo(50_000)
        outputs = [
            {'script': '51', 'satoshis': 10_000},
            {'script': '51', 'satoshis': 10_000},
        ]

        tx_hex, input_count, change_amount = build_call_transaction(
            current_utxo=utxo,
            unlocking_script='51',
            new_locking_script='',  # Empty since contract_outputs is used
            new_satoshis=0,
            change_address='00' * 20,
            contract_outputs=outputs,
        )

        assert input_count == 1
        assert isinstance(tx_hex, str)
        assert all(c in '0123456789abcdef' for c in tx_hex)

    def test_change_amount_is_non_negative(self):
        """Change amount should never be negative."""
        utxo = _make_utxo(100_000)
        _, _, change_amount = build_call_transaction(
            current_utxo=utxo,
            unlocking_script='51',
            new_locking_script='51',
            new_satoshis=10_000,
            change_address='00' * 20,
        )
        assert change_amount >= 0


# ---------------------------------------------------------------------------
# insert_unlocking_script
# ---------------------------------------------------------------------------

class TestInsertUnlockingScript:
    def test_replaces_empty_scriptsig(self):
        """Inserting an unlocking script into an unsigned input replaces the empty scriptSig."""
        utxo = _make_utxo(50_000)
        tx_hex, _, _ = build_call_transaction(
            current_utxo=utxo,
            unlocking_script='51',  # 1-byte unlock
            new_locking_script='51',
            new_satoshis=10_000,
            change_address='00' * 20,
        )

        new_unlock = 'aabb'  # 2 bytes
        modified = insert_unlocking_script(tx_hex, 0, new_unlock)

        assert isinstance(modified, str)
        assert all(c in '0123456789abcdef' for c in modified)
        # The new unlock script should appear somewhere in the output
        assert new_unlock in modified

    def test_out_of_range_index_raises(self):
        """Inserting at an invalid input index raises ValueError."""
        utxo = _make_utxo(50_000)
        tx_hex, _, _ = build_call_transaction(
            current_utxo=utxo,
            unlocking_script='51',
            new_locking_script='51',
            new_satoshis=10_000,
            change_address='00' * 20,
        )

        with pytest.raises(ValueError, match='out of range'):
            insert_unlocking_script(tx_hex, 5, 'aabb')
