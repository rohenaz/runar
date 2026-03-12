"""Tests for runar.types and runar.sdk.types — type definitions and artifact loading."""

import pytest
from runar.types import Readonly
from runar.sdk.types import (
    RunarArtifact, Abi, AbiParam, AbiMethod, StateField,
    ConstructorSlot, Utxo, TransactionData, TxInput, TxOutput,
    DeployOptions, CallOptions, TerminalOutput,
)


# ---------------------------------------------------------------------------
# RunarArtifact.from_dict
# ---------------------------------------------------------------------------

class TestRunarArtifactFromDict:
    def test_loads_full_artifact(self):
        """from_dict correctly parses a complete artifact dict."""
        raw = {
            'version': 'runar-v0.1.0',
            'compilerVersion': '1.2.3',
            'contractName': 'Counter',
            'abi': {
                'constructor': {
                    'params': [
                        {'name': 'owner', 'type': 'PubKey'},
                    ],
                },
                'methods': [
                    {
                        'name': 'increment',
                        'params': [{'name': 'amount', 'type': 'bigint'}],
                        'isPublic': True,
                    },
                    {
                        'name': 'reset',
                        'params': [],
                        'isPublic': True,
                    },
                ],
            },
            'script': '51516a',
            'asm': 'OP_TRUE OP_TRUE OP_RETURN',
            'stateFields': [
                {'name': 'count', 'type': 'bigint', 'index': 0, 'initialValue': '0n'},
            ],
            'constructorSlots': [
                {'paramIndex': 0, 'byteOffset': 10},
            ],
            'buildTimestamp': '2025-01-01T00:00:00Z',
            'codeSeparatorIndex': 5,
            'codeSeparatorIndices': [5, 20],
        }

        artifact = RunarArtifact.from_dict(raw)

        assert artifact.version == 'runar-v0.1.0'
        assert artifact.compiler_version == '1.2.3'
        assert artifact.contract_name == 'Counter'
        assert artifact.script == '51516a'
        assert artifact.asm == 'OP_TRUE OP_TRUE OP_RETURN'
        assert artifact.build_timestamp == '2025-01-01T00:00:00Z'
        assert artifact.code_separator_index == 5
        assert artifact.code_separator_indices == [5, 20]

        # ABI
        assert len(artifact.abi.constructor_params) == 1
        assert artifact.abi.constructor_params[0].name == 'owner'
        assert artifact.abi.constructor_params[0].type == 'PubKey'
        assert len(artifact.abi.methods) == 2
        assert artifact.abi.methods[0].name == 'increment'
        assert len(artifact.abi.methods[0].params) == 1
        assert artifact.abi.methods[1].name == 'reset'

    def test_handles_state_fields(self):
        raw = {
            'version': 'runar-v0.1.0',
            'contractName': 'Test',
            'abi': {'constructor': {'params': []}, 'methods': []},
            'script': '51',
            'stateFields': [
                {'name': 'a', 'type': 'bigint', 'index': 0, 'initialValue': '5n'},
                {'name': 'b', 'type': 'bool', 'index': 1},
                {'name': 'c', 'type': 'PubKey', 'index': 2},
            ],
        }

        artifact = RunarArtifact.from_dict(raw)
        assert len(artifact.state_fields) == 3
        assert artifact.state_fields[0].name == 'a'
        assert artifact.state_fields[0].initial_value == '5n'
        assert artifact.state_fields[1].name == 'b'
        assert artifact.state_fields[1].initial_value is None
        assert artifact.state_fields[2].type == 'PubKey'

    def test_handles_constructor_slots(self):
        raw = {
            'version': 'runar-v0.1.0',
            'contractName': 'Test',
            'abi': {'constructor': {'params': []}, 'methods': []},
            'script': '51',
            'constructorSlots': [
                {'paramIndex': 0, 'byteOffset': 5},
                {'paramIndex': 1, 'byteOffset': 40},
            ],
        }

        artifact = RunarArtifact.from_dict(raw)
        assert len(artifact.constructor_slots) == 2
        assert artifact.constructor_slots[0].param_index == 0
        assert artifact.constructor_slots[0].byte_offset == 5
        assert artifact.constructor_slots[1].param_index == 1
        assert artifact.constructor_slots[1].byte_offset == 40

    def test_defaults_for_missing_fields(self):
        """Minimal dict still produces a valid artifact."""
        raw = {}
        artifact = RunarArtifact.from_dict(raw)
        assert artifact.version == ''
        assert artifact.contract_name == ''
        assert artifact.script == ''
        assert artifact.state_fields == []
        assert artifact.constructor_slots == []


# ---------------------------------------------------------------------------
# Dataclass creation
# ---------------------------------------------------------------------------

class TestDataclasses:
    def test_utxo_creation(self):
        u = Utxo(txid='aa' * 32, output_index=1, satoshis=50000, script='51')
        assert u.txid == 'aa' * 32
        assert u.output_index == 1
        assert u.satoshis == 50000
        assert u.script == '51'

    def test_tx_input(self):
        inp = TxInput(txid='bb' * 32, output_index=0, script='5151')
        assert inp.sequence == 0xFFFFFFFF  # default

    def test_tx_output(self):
        out = TxOutput(satoshis=10000, script='76a914' + '00' * 20 + '88ac')
        assert out.satoshis == 10000

    def test_transaction_data_defaults(self):
        tx = TransactionData(txid='cc' * 32)
        assert tx.version == 1
        assert tx.inputs == []
        assert tx.outputs == []
        assert tx.locktime == 0

    def test_deploy_options_defaults(self):
        opts = DeployOptions()
        assert opts.satoshis == 10000
        assert opts.change_address == ''

    def test_terminal_output(self):
        to = TerminalOutput(script_hex='51', satoshis=5000)
        assert to.script_hex == '51'
        assert to.satoshis == 5000

    def test_constructor_slot(self):
        cs = ConstructorSlot(param_index=0, byte_offset=42)
        assert cs.param_index == 0
        assert cs.byte_offset == 42


# ---------------------------------------------------------------------------
# Readonly type
# ---------------------------------------------------------------------------

class TestReadonly:
    def test_readonly_type_annotation(self):
        """Readonly[int] can be used as a type annotation without error."""
        # This just verifies the generic type resolves; it's a marker type
        annotation = Readonly[int]
        assert annotation is not None

    def test_readonly_bytes(self):
        annotation = Readonly[bytes]
        assert annotation is not None
