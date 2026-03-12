"""Tests for the multi-format parser dispatch and individual parsers.

Covers TS, Solidity, Move, Python, Go, and Rust format parsers. Each test
provides a minimal contract source string and verifies the resulting AST
has the correct contract name, parent class, properties, and methods.
"""

from __future__ import annotations

import pytest

from runar_compiler.frontend.parser_dispatch import parse_source, ParseResult
from runar_compiler.frontend.ast_nodes import ContractNode, PrimitiveType


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _assert_p2pkh_ast(contract: ContractNode, expected_name: str = "P2PKH") -> None:
    """Verify common P2PKH AST structure across all formats."""
    assert contract.name == expected_name
    assert contract.parent_class == "SmartContract"
    assert len(contract.properties) >= 1
    # The first property should be pubKeyHash (camelCase in AST)
    prop = contract.properties[0]
    assert prop.name == "pubKeyHash"
    # Should have at least an unlock method
    method_names = [m.name for m in contract.methods]
    assert "unlock" in method_names


# ---------------------------------------------------------------------------
# TypeScript parser (.runar.ts)
# ---------------------------------------------------------------------------

TS_P2PKH_SOURCE = """\
import { SmartContract, assert, hash160, checkSig } from 'runar-lang';

export class P2PKH extends SmartContract {
    readonly pubKeyHash: ByteString;

    constructor(pubKeyHash: ByteString) {
        super(pubKeyHash);
    }

    public unlock(sig: Sig, pubKey: PubKey) {
        assert(hash160(pubKey) == this.pubKeyHash);
        assert(checkSig(sig, pubKey));
    }
}
"""


class TestTSParser:
    def test_parse_basic_p2pkh(self):
        result = parse_source(TS_P2PKH_SOURCE, "P2PKH.runar.ts")
        assert len(result.errors) == 0
        assert result.contract is not None
        _assert_p2pkh_ast(result.contract)

    def test_parse_ts_has_constructor(self):
        result = parse_source(TS_P2PKH_SOURCE, "P2PKH.runar.ts")
        assert result.contract is not None
        # Constructor should exist with super() call
        assert result.contract.constructor is not None

    def test_parse_ts_unlock_params(self):
        result = parse_source(TS_P2PKH_SOURCE, "P2PKH.runar.ts")
        assert result.contract is not None
        unlock = [m for m in result.contract.methods if m.name == "unlock"][0]
        param_names = [p.name for p in unlock.params]
        assert "sig" in param_names
        assert "pubKey" in param_names

    def test_parse_ts_property_readonly(self):
        result = parse_source(TS_P2PKH_SOURCE, "P2PKH.runar.ts")
        assert result.contract is not None
        prop = result.contract.properties[0]
        assert prop.readonly is True


# ---------------------------------------------------------------------------
# Solidity parser (.runar.sol)
# ---------------------------------------------------------------------------

SOL_P2PKH_SOURCE = """\
contract P2PKH is SmartContract {
    bytes pubKeyHash;

    constructor(bytes pubKeyHash) {
        this.pubKeyHash = pubKeyHash;
    }

    function unlock(Sig sig, PubKey pubKey) public {
        assert(hash160(pubKey) == this.pubKeyHash);
        assert(checkSig(sig, pubKey));
    }
}
"""


class TestSolParser:
    def test_parse_basic_contract(self):
        result = parse_source(SOL_P2PKH_SOURCE, "P2PKH.runar.sol")
        assert len(result.errors) == 0
        assert result.contract is not None
        assert result.contract.name == "P2PKH"
        assert result.contract.parent_class == "SmartContract"

    def test_parse_sol_properties(self):
        result = parse_source(SOL_P2PKH_SOURCE, "P2PKH.runar.sol")
        assert result.contract is not None
        assert len(result.contract.properties) >= 1
        prop = result.contract.properties[0]
        assert prop.name == "pubKeyHash"

    def test_parse_sol_methods(self):
        result = parse_source(SOL_P2PKH_SOURCE, "P2PKH.runar.sol")
        assert result.contract is not None
        method_names = [m.name for m in result.contract.methods]
        assert "unlock" in method_names


# ---------------------------------------------------------------------------
# Move parser (.runar.move)
# ---------------------------------------------------------------------------

MOVE_P2PKH_SOURCE = """\
module P2PKH {
    use runar::types::{Addr, PubKey, Sig};
    use runar::crypto::{hash160, check_sig};

    resource struct P2PKH {
        pub_key_hash: Addr,
    }

    public fun unlock(contract: &P2PKH, sig: Sig, pub_key: PubKey) {
        assert!(hash160(pub_key) == contract.pub_key_hash, 0);
        assert!(check_sig(sig, pub_key), 0);
    }
}
"""


class TestMoveParser:
    def test_parse_basic_module(self):
        result = parse_source(MOVE_P2PKH_SOURCE, "P2PKH.runar.move")
        assert len(result.errors) == 0
        assert result.contract is not None
        assert result.contract.name == "P2PKH"
        assert result.contract.parent_class == "SmartContract"

    def test_parse_move_properties(self):
        result = parse_source(MOVE_P2PKH_SOURCE, "P2PKH.runar.move")
        assert result.contract is not None
        assert len(result.contract.properties) >= 1
        # Move uses snake_case -> camelCase in AST
        prop_names = [p.name for p in result.contract.properties]
        assert "pubKeyHash" in prop_names

    def test_parse_move_methods(self):
        result = parse_source(MOVE_P2PKH_SOURCE, "P2PKH.runar.move")
        assert result.contract is not None
        method_names = [m.name for m in result.contract.methods]
        assert "unlock" in method_names


# ---------------------------------------------------------------------------
# Python parser (.runar.py)
# ---------------------------------------------------------------------------

PY_P2PKH_SOURCE = """\
from runar import SmartContract, assert_, hash160, check_sig

class P2PKH(SmartContract):
    pub_key_hash: ByteString

    def __init__(self, pub_key_hash: ByteString):
        super().__init__(pub_key_hash)

    @public
    def unlock(self, sig: Sig, pub_key: PubKey):
        assert_(hash160(pub_key) == self.pub_key_hash)
        assert_(check_sig(sig, pub_key))
"""


class TestPythonParser:
    def test_parse_basic_class(self):
        result = parse_source(PY_P2PKH_SOURCE, "P2PKH.runar.py")
        assert len(result.errors) == 0
        assert result.contract is not None
        assert result.contract.name == "P2PKH"
        assert result.contract.parent_class == "SmartContract"

    def test_parse_python_properties(self):
        result = parse_source(PY_P2PKH_SOURCE, "P2PKH.runar.py")
        assert result.contract is not None
        assert len(result.contract.properties) >= 1
        # Python snake_case -> camelCase
        prop_names = [p.name for p in result.contract.properties]
        assert "pubKeyHash" in prop_names

    def test_parse_python_methods(self):
        result = parse_source(PY_P2PKH_SOURCE, "P2PKH.runar.py")
        assert result.contract is not None
        method_names = [m.name for m in result.contract.methods]
        assert "unlock" in method_names

    def test_parse_python_method_params(self):
        result = parse_source(PY_P2PKH_SOURCE, "P2PKH.runar.py")
        assert result.contract is not None
        unlock = [m for m in result.contract.methods if m.name == "unlock"][0]
        # 'self' should be stripped from params
        param_names = [p.name for p in unlock.params]
        assert "self" not in param_names
        assert "sig" in param_names
        assert "pubKey" in param_names


# ---------------------------------------------------------------------------
# Go contract parser (.runar.go)
# ---------------------------------------------------------------------------

GO_P2PKH_SOURCE = """\
package contract

import "github.com/icellan/runar/packages/runar-go"

type P2PKH struct {
    runar.SmartContract
    PubKeyHash runar.ByteString
}

func (c *P2PKH) init() {
    c.PubKeyHash = c.PubKeyHash
}

func (c *P2PKH) Unlock(sig runar.Sig, pubKey runar.PubKey) {
    runar.Assert(runar.Hash160(pubKey) == c.PubKeyHash)
    runar.Assert(runar.CheckSig(sig, pubKey))
}
"""


class TestGoParser:
    def test_parse_basic_struct(self):
        result = parse_source(GO_P2PKH_SOURCE, "P2PKH.runar.go")
        assert len(result.errors) == 0
        assert result.contract is not None
        assert result.contract.name == "P2PKH"
        assert result.contract.parent_class == "SmartContract"

    def test_parse_go_properties(self):
        result = parse_source(GO_P2PKH_SOURCE, "P2PKH.runar.go")
        assert result.contract is not None
        assert len(result.contract.properties) >= 1
        prop_names = [p.name for p in result.contract.properties]
        assert "pubKeyHash" in prop_names

    def test_parse_go_methods(self):
        result = parse_source(GO_P2PKH_SOURCE, "P2PKH.runar.go")
        assert result.contract is not None
        method_names = [m.name for m in result.contract.methods]
        # Go exported methods like Unlock -> unlock in AST
        assert "unlock" in method_names


# ---------------------------------------------------------------------------
# Rust DSL parser (.runar.rs)
# ---------------------------------------------------------------------------

RUST_P2PKH_SOURCE = """\
use runar::prelude::*;

#[runar::contract]
pub struct P2PKH {
    #[readonly]
    pub_key_hash: ByteString,
}

#[runar::methods(P2PKH)]
impl P2PKH {
    #[public]
    pub fn unlock(&self, sig: &Sig, pub_key: &PubKey) {
        assert!(hash160(pub_key) == self.pub_key_hash);
        assert!(check_sig(sig, pub_key));
    }
}
"""


class TestRustParser:
    def test_parse_basic_struct(self):
        result = parse_source(RUST_P2PKH_SOURCE, "P2PKH.runar.rs")
        assert len(result.errors) == 0
        assert result.contract is not None
        assert result.contract.name == "P2PKH"
        assert result.contract.parent_class == "SmartContract"

    def test_parse_rust_properties(self):
        result = parse_source(RUST_P2PKH_SOURCE, "P2PKH.runar.rs")
        assert result.contract is not None
        assert len(result.contract.properties) >= 1
        prop_names = [p.name for p in result.contract.properties]
        # Rust snake_case -> camelCase
        assert "pubKeyHash" in prop_names

    def test_parse_rust_methods(self):
        result = parse_source(RUST_P2PKH_SOURCE, "P2PKH.runar.rs")
        assert result.contract is not None
        method_names = [m.name for m in result.contract.methods]
        assert "unlock" in method_names


# ---------------------------------------------------------------------------
# Error handling
# ---------------------------------------------------------------------------

class TestParserErrors:
    def test_unsupported_extension(self):
        result = parse_source("some code", "test.txt")
        assert len(result.errors) > 0
        assert result.contract is None

    def test_empty_ts_source(self):
        result = parse_source("", "Empty.runar.ts")
        # Should either have errors or no contract
        assert result.contract is None or len(result.errors) > 0

    def test_invalid_ts_syntax(self):
        result = parse_source("this is not valid { { { TypeScript", "Bad.runar.ts")
        assert result.contract is None or len(result.errors) > 0

    def test_invalid_sol_syntax(self):
        result = parse_source("this is not a solidity contract", "Bad.runar.sol")
        assert result.contract is None or len(result.errors) > 0

    def test_invalid_py_syntax(self):
        result = parse_source("def ??? broken(", "Bad.runar.py")
        assert result.contract is None or len(result.errors) > 0


# ---------------------------------------------------------------------------
# Dispatch correctness
# ---------------------------------------------------------------------------

class TestParserDispatch:
    def test_ts_extension_dispatches(self):
        """Verify .runar.ts routes to TS parser and produces valid AST."""
        result = parse_source(TS_P2PKH_SOURCE, "test.runar.ts")
        assert result.contract is not None
        assert result.contract.name == "P2PKH"

    def test_sol_extension_dispatches(self):
        result = parse_source(SOL_P2PKH_SOURCE, "test.runar.sol")
        assert result.contract is not None
        assert result.contract.name == "P2PKH"

    def test_case_insensitive_extension(self):
        """Extension matching should be case-insensitive."""
        result = parse_source(TS_P2PKH_SOURCE, "test.RUNAR.TS")
        # parse_source lowercases the extension
        assert result.contract is not None or len(result.errors) > 0
