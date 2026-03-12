"""Tests for runar.sdk.state — state serialization and deserialization."""

import pytest
from runar.sdk.state import (
    serialize_state, deserialize_state, extract_state_from_script,
    find_last_op_return, encode_push_data, decode_push_data,
)
from runar.sdk.types import StateField, RunarArtifact, Abi, AbiMethod


def _make_artifact(state_fields: list[StateField], script: str = '51') -> RunarArtifact:
    return RunarArtifact(
        version='runar-v0.1.0',
        contract_name='Test',
        abi=Abi(constructor_params=[], methods=[]),
        script=script,
        state_fields=state_fields,
    )


# ---------------------------------------------------------------------------
# serialize_state / deserialize_state round-trip
# ---------------------------------------------------------------------------

class TestStateRoundTrip:
    def test_bigint_round_trip(self):
        """Serializing and deserializing a bigint value produces the same integer."""
        fields = [StateField(name='count', type='bigint', index=0)]
        values = {'count': 42}
        hex_out = serialize_state(fields, values)
        result = deserialize_state(fields, hex_out)
        assert result['count'] == 42

    def test_bigint_zero_round_trip(self):
        fields = [StateField(name='val', type='bigint', index=0)]
        hex_out = serialize_state(fields, {'val': 0})
        result = deserialize_state(fields, hex_out)
        assert result['val'] == 0

    def test_bigint_negative_round_trip(self):
        fields = [StateField(name='val', type='bigint', index=0)]
        hex_out = serialize_state(fields, {'val': -999})
        result = deserialize_state(fields, hex_out)
        assert result['val'] == -999

    def test_bool_true_round_trip(self):
        fields = [StateField(name='flag', type='bool', index=0)]
        hex_out = serialize_state(fields, {'flag': True})
        result = deserialize_state(fields, hex_out)
        assert result['flag'] is True

    def test_bool_false_round_trip(self):
        fields = [StateField(name='flag', type='bool', index=0)]
        hex_out = serialize_state(fields, {'flag': False})
        result = deserialize_state(fields, hex_out)
        assert result['flag'] is False

    def test_pubkey_round_trip(self):
        """PubKey (33 bytes) round-trips as raw hex without push-data prefix."""
        fields = [StateField(name='pk', type='PubKey', index=0)]
        pubkey_hex = '02' + 'ab' * 32  # 33 bytes
        hex_out = serialize_state(fields, {'pk': pubkey_hex})
        result = deserialize_state(fields, hex_out)
        assert result['pk'] == pubkey_hex

    def test_multiple_fields_ordered_by_index(self):
        """Fields are serialized in index order regardless of dict insertion order."""
        fields = [
            StateField(name='b', type='bigint', index=1),
            StateField(name='a', type='bigint', index=0),
        ]
        values = {'a': 10, 'b': 20}
        hex_out = serialize_state(fields, values)
        result = deserialize_state(fields, hex_out)
        assert result['a'] == 10
        assert result['b'] == 20


# ---------------------------------------------------------------------------
# find_last_op_return
# ---------------------------------------------------------------------------

class TestFindLastOpReturn:
    def test_finds_op_return_in_script(self):
        """OP_RETURN (0x6a) at a real opcode boundary is found."""
        # OP_TRUE (0x51) + OP_RETURN (0x6a) + some data
        script = '51' + '6a' + 'deadbeef'
        pos = find_last_op_return(script)
        assert pos == 2  # hex-char offset

    def test_returns_negative_one_when_absent(self):
        """No OP_RETURN in the script returns -1."""
        script = '5151'  # OP_TRUE OP_TRUE
        pos = find_last_op_return(script)
        assert pos == -1

    def test_skips_0x6a_inside_push_data(self):
        """0x6a embedded inside a push data segment should not be mistaken for OP_RETURN."""
        # Push 2 bytes: 0x02 0x6a 0x6a (the pushed data contains 0x6a)
        # Then OP_TRUE
        script = '026a6a' + '51'
        pos = find_last_op_return(script)
        # The 0x6a bytes are inside push data, so no real OP_RETURN
        assert pos == -1


# ---------------------------------------------------------------------------
# extract_state_from_script
# ---------------------------------------------------------------------------

class TestExtractStateFromScript:
    def test_extracts_state_after_op_return(self):
        """State is everything after the last OP_RETURN."""
        fields = [StateField(name='count', type='bigint', index=0)]
        artifact = _make_artifact(fields)

        # Build a script: OP_TRUE + OP_RETURN + serialized state
        state_hex = serialize_state(fields, {'count': 7})
        full_script = '51' + '6a' + state_hex

        result = extract_state_from_script(artifact, full_script)
        assert result is not None
        assert result['count'] == 7

    def test_returns_none_without_state_fields(self):
        artifact = _make_artifact([])
        result = extract_state_from_script(artifact, '516adeadbeef')
        assert result is None

    def test_returns_none_without_op_return(self):
        fields = [StateField(name='count', type='bigint', index=0)]
        artifact = _make_artifact(fields)
        result = extract_state_from_script(artifact, '5151')
        assert result is None


# ---------------------------------------------------------------------------
# encode_push_data / decode_push_data round-trip
# ---------------------------------------------------------------------------

class TestPushDataRoundTrip:
    def test_small_data(self):
        """Data <= 75 bytes uses direct length prefix."""
        data_hex = 'aabbccdd'  # 4 bytes
        encoded = encode_push_data(data_hex)
        decoded, consumed = decode_push_data(encoded, 0)
        assert decoded == data_hex
        assert consumed == len(encoded)

    def test_single_byte(self):
        data_hex = 'ff'
        encoded = encode_push_data(data_hex)
        decoded, _ = decode_push_data(encoded, 0)
        assert decoded == data_hex

    def test_76_bytes_uses_op_pushdata1(self):
        """Data of 76 bytes triggers OP_PUSHDATA1 (0x4c) prefix."""
        data_hex = 'ab' * 76
        encoded = encode_push_data(data_hex)
        # Should start with 0x4c (OP_PUSHDATA1) then length byte
        assert encoded[:2] == '4c'
        decoded, consumed = decode_push_data(encoded, 0)
        assert decoded == data_hex
        assert consumed == len(encoded)

    def test_empty_data(self):
        data_hex = ''
        encoded = encode_push_data(data_hex)
        decoded, consumed = decode_push_data(encoded, 0)
        assert decoded == data_hex
