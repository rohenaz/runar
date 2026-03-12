"""Tests for runar.sdk.rpc_provider — RPCProvider with mocked HTTP."""

import json
import pytest
from unittest.mock import patch, MagicMock, PropertyMock
from runar.sdk.rpc_provider import RPCProvider


# ---------------------------------------------------------------------------
# Constructor / basic properties
# ---------------------------------------------------------------------------

class TestRPCProviderInit:
    def test_stores_url_and_credentials(self):
        provider = RPCProvider('http://localhost:8332', 'alice', 'secret')
        assert provider._url == 'http://localhost:8332'
        # Auth is base64('alice:secret')
        import base64
        expected_auth = base64.b64encode(b'alice:secret').decode()
        assert provider._auth == expected_auth

    def test_get_network_default(self):
        provider = RPCProvider('http://localhost:8332', 'user', 'pass')
        assert provider.get_network() == 'testnet'

    def test_get_network_custom(self):
        provider = RPCProvider('http://localhost:8332', 'user', 'pass', network='mainnet')
        assert provider.get_network() == 'mainnet'

    def test_get_fee_rate_default(self):
        provider = RPCProvider('http://localhost:8332', 'user', 'pass')
        assert provider.get_fee_rate() == 1


# ---------------------------------------------------------------------------
# regtest classmethod
# ---------------------------------------------------------------------------

class TestRegtestFactory:
    def test_regtest_creates_correct_provider(self):
        provider = RPCProvider.regtest('http://localhost:18332', 'bitcoin', 'bitcoin')
        assert provider._url == 'http://localhost:18332'
        assert provider._auto_mine is True
        assert provider.get_network() == 'regtest'


# ---------------------------------------------------------------------------
# RPC call methods (mocked)
# ---------------------------------------------------------------------------

class TestRPCCalls:
    def _mock_rpc(self, provider, return_value):
        """Patch _rpc_call to return a fixed value."""
        return patch.object(provider, '_rpc_call', return_value=return_value)

    def test_get_transaction_calls_getrawtransaction(self):
        provider = RPCProvider('http://localhost:8332', 'user', 'pass')
        fake_result = {
            'hex': 'deadbeef',
            'vout': [
                {'value': 0.0001, 'scriptPubKey': {'hex': '51'}},
            ],
        }
        with patch.object(provider, '_rpc_call', return_value=fake_result) as mock:
            tx = provider.get_transaction('aa' * 32)
            mock.assert_called_once_with('getrawtransaction', 'aa' * 32, True)

        assert tx.txid == 'aa' * 32
        assert tx.raw == 'deadbeef'
        assert len(tx.outputs) == 1
        assert tx.outputs[0].satoshis == 10000
        assert tx.outputs[0].script == '51'

    def test_broadcast_calls_sendrawtransaction(self):
        provider = RPCProvider('http://localhost:8332', 'user', 'pass')
        fake_txid = 'bb' * 32
        with patch.object(provider, '_rpc_call', return_value=fake_txid) as mock:
            txid = provider.broadcast('01000000...')
            mock.assert_called_once_with('sendrawtransaction', '01000000...')

        assert txid == fake_txid

    def test_broadcast_auto_mines_when_enabled(self):
        provider = RPCProvider.regtest('http://localhost:18332', 'bitcoin', 'bitcoin')
        call_results = iter(['cc' * 32, None])  # sendrawtransaction, then generate

        with patch.object(provider, '_rpc_call', side_effect=call_results) as mock:
            txid = provider.broadcast('deadbeef')

        assert mock.call_count == 2
        # Second call should be 'generate'
        assert mock.call_args_list[1][0][0] == 'generate'

    def test_get_utxos_parses_result(self):
        provider = RPCProvider('http://localhost:8332', 'user', 'pass')
        fake_result = [
            {
                'txid': 'dd' * 32,
                'vout': 0,
                'amount': 0.5,
                'scriptPubKey': '76a914' + '00' * 20 + '88ac',
            },
            {
                'txid': 'ee' * 32,
                'vout': 1,
                'amount': 1.0,
                'scriptPubKey': '76a914' + 'ff' * 20 + '88ac',
            },
        ]
        with patch.object(provider, '_rpc_call', return_value=fake_result) as mock:
            utxos = provider.get_utxos('1SomeAddress')
            mock.assert_called_once_with('listunspent', 0, 9999999, ['1SomeAddress'])

        assert len(utxos) == 2
        assert utxos[0].txid == 'dd' * 32
        assert utxos[0].output_index == 0
        assert utxos[0].satoshis == 50_000_000
        assert utxos[1].satoshis == 100_000_000

    def test_get_raw_transaction(self):
        provider = RPCProvider('http://localhost:8332', 'user', 'pass')
        with patch.object(provider, '_rpc_call', return_value='cafebabe') as mock:
            raw = provider.get_raw_transaction('ff' * 32)
            mock.assert_called_once_with('getrawtransaction', 'ff' * 32, False)
        assert raw == 'cafebabe'

    def test_get_contract_utxo_returns_none(self):
        """RPCProvider.get_contract_utxo always returns None (not supported via RPC)."""
        provider = RPCProvider('http://localhost:8332', 'user', 'pass')
        assert provider.get_contract_utxo('somehash') is None
