"""Tests for the ANF EC optimizer (Pass 4.5).

Covers algebraic simplification rules for EC intrinsic calls, pass-through
behavior for non-EC programs, and dead binding elimination.
"""

from __future__ import annotations

import pytest

from runar_compiler.frontend.anf_optimize import (
    optimize_ec,
    INFINITY_HEX,
    G_HEX,
    CURVE_N,
)
from runar_compiler.ir.types import (
    ANFBinding,
    ANFMethod,
    ANFParam,
    ANFProgram,
    ANFProperty,
    ANFValue,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_program(bindings: list[ANFBinding]) -> ANFProgram:
    """Create a minimal program with a single method containing the given bindings."""
    return ANFProgram(
        contract_name="Test",
        properties=[],
        methods=[
            ANFMethod(
                name="test",
                params=[],
                body=bindings,
                is_public=True,
            ),
        ],
    )


def _load_const_hex(name: str, hex_str: str) -> ANFBinding:
    return ANFBinding(
        name=name,
        value=ANFValue(kind="load_const", const_string=hex_str, raw_value=hex_str),
    )


def _load_const_int(name: str, n: int) -> ANFBinding:
    return ANFBinding(
        name=name,
        value=ANFValue(kind="load_const", const_big_int=n, const_int=n, raw_value=n),
    )


def _call(name: str, func: str, args: list[str]) -> ANFBinding:
    return ANFBinding(
        name=name,
        value=ANFValue(kind="call", func=func, args=args),
    )


def _assert_ref(name: str, value_ref: str) -> ANFBinding:
    return ANFBinding(
        name=name,
        value=ANFValue(kind="assert", value_ref=value_ref, raw_value=value_ref),
    )


def _get_method_body(program: ANFProgram) -> list[ANFBinding]:
    return program.methods[0].body


def _find_binding(bindings: list[ANFBinding], name: str) -> ANFBinding | None:
    for b in bindings:
        if b.name == name:
            return b
    return None


# ---------------------------------------------------------------------------
# Pass-through behavior (no EC ops)
# ---------------------------------------------------------------------------

class TestPassThrough:
    def test_no_ec_ops_unchanged(self):
        """Programs without EC calls pass through without modification."""
        bindings = [
            _load_const_int("t0", 42),
            _load_const_int("t1", 10),
            ANFBinding(
                name="t2",
                value=ANFValue(kind="bin_op", op="+", left="t0", right="t1"),
            ),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)

        body = _get_method_body(result)
        assert len(body) == 4
        assert body[0].name == "t0"
        assert body[1].name == "t1"
        assert body[2].name == "t2"
        assert body[3].name == "t3"

    def test_empty_method_unchanged(self):
        program = _make_program([])
        result = optimize_ec(program)
        assert len(_get_method_body(result)) == 0

    def test_non_ec_call_unchanged(self):
        """Calls to non-EC functions (e.g. hash160) are not optimized."""
        bindings = [
            _load_const_hex("t0", "abcd"),
            _call("t1", "hash160", ["t0"]),
            _assert_ref("t2", "t1"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)
        assert len(body) == 3


# ---------------------------------------------------------------------------
# Rule 1: ecAdd(x, INFINITY) -> x
# ---------------------------------------------------------------------------

class TestRule1EcAddInfinity:
    def test_ec_add_x_infinity(self):
        bindings = [
            _load_const_hex("t0", "ab" * 64),  # some point
            _load_const_hex("t1", INFINITY_HEX),
            _call("t2", "ecAdd", ["t0", "t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        # t2 should become a @ref: to t0
        t2 = _find_binding(body, "t2")
        assert t2 is not None
        assert t2.value.kind == "load_param"
        assert t2.value.name == "@ref:t0"


# ---------------------------------------------------------------------------
# Rule 2: ecAdd(INFINITY, x) -> x
# ---------------------------------------------------------------------------

class TestRule2EcAddInfinityLeft:
    def test_ec_add_infinity_x(self):
        bindings = [
            _load_const_hex("t0", INFINITY_HEX),
            _load_const_hex("t1", "cd" * 64),  # some point
            _call("t2", "ecAdd", ["t0", "t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t2 = _find_binding(body, "t2")
        assert t2 is not None
        assert t2.value.kind == "load_param"
        assert t2.value.name == "@ref:t1"


# ---------------------------------------------------------------------------
# Rule 3: ecMul(x, 1) -> x
# ---------------------------------------------------------------------------

class TestRule3EcMulByOne:
    def test_ec_mul_x_1(self):
        bindings = [
            _load_const_hex("t0", "ab" * 64),
            _load_const_int("t1", 1),
            _call("t2", "ecMul", ["t0", "t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t2 = _find_binding(body, "t2")
        assert t2 is not None
        assert t2.value.kind == "load_param"
        assert t2.value.name == "@ref:t0"


# ---------------------------------------------------------------------------
# Rule 4: ecMul(x, 0) -> INFINITY
# ---------------------------------------------------------------------------

class TestRule4EcMulByZero:
    def test_ec_mul_x_0(self):
        bindings = [
            _load_const_hex("t0", "ab" * 64),
            _load_const_int("t1", 0),
            _call("t2", "ecMul", ["t0", "t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t2 = _find_binding(body, "t2")
        assert t2 is not None
        assert t2.value.kind == "load_const"
        assert t2.value.const_string == INFINITY_HEX


# ---------------------------------------------------------------------------
# Rule 5: ecMulGen(0) -> INFINITY
# ---------------------------------------------------------------------------

class TestRule5EcMulGenZero:
    def test_ec_mulgen_0(self):
        bindings = [
            _load_const_int("t0", 0),
            _call("t1", "ecMulGen", ["t0"]),
            _assert_ref("t2", "t1"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t1 = _find_binding(body, "t1")
        assert t1 is not None
        assert t1.value.kind == "load_const"
        assert t1.value.const_string == INFINITY_HEX


# ---------------------------------------------------------------------------
# Rule 6: ecMulGen(1) -> G
# ---------------------------------------------------------------------------

class TestRule6EcMulGenOne:
    def test_ec_mulgen_1(self):
        bindings = [
            _load_const_int("t0", 1),
            _call("t1", "ecMulGen", ["t0"]),
            _assert_ref("t2", "t1"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t1 = _find_binding(body, "t1")
        assert t1 is not None
        assert t1.value.kind == "load_const"
        assert t1.value.const_string == G_HEX


# ---------------------------------------------------------------------------
# Rule 7: ecNegate(ecNegate(x)) -> x
# ---------------------------------------------------------------------------

class TestRule7DoubleNegate:
    def test_double_negate(self):
        bindings = [
            _load_const_hex("t0", "ab" * 64),
            _call("t1", "ecNegate", ["t0"]),
            _call("t2", "ecNegate", ["t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t2 = _find_binding(body, "t2")
        assert t2 is not None
        assert t2.value.kind == "load_param"
        assert t2.value.name == "@ref:t0"


# ---------------------------------------------------------------------------
# Rule 8: ecAdd(x, ecNegate(x)) -> INFINITY
# ---------------------------------------------------------------------------

class TestRule8AddNegate:
    def test_add_self_negate(self):
        bindings = [
            _load_const_hex("t0", "ab" * 64),
            _call("t1", "ecNegate", ["t0"]),
            _call("t2", "ecAdd", ["t0", "t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t2 = _find_binding(body, "t2")
        assert t2 is not None
        assert t2.value.kind == "load_const"
        assert t2.value.const_string == INFINITY_HEX


# ---------------------------------------------------------------------------
# Rule 12: ecMul(G, k) -> ecMulGen(k)
# ---------------------------------------------------------------------------

class TestRule12MulGToMulGen:
    def test_mul_g_k(self):
        bindings = [
            _load_const_hex("t0", G_HEX),
            _load_const_int("t1", 42),
            _call("t2", "ecMul", ["t0", "t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        t2 = _find_binding(body, "t2")
        assert t2 is not None
        assert t2.value.kind == "call"
        assert t2.value.func == "ecMulGen"
        assert t2.value.args == ["t1"]


# ---------------------------------------------------------------------------
# Dead binding elimination
# ---------------------------------------------------------------------------

class TestDeadBindingElimination:
    def test_dead_binding_removed(self):
        """A binding not referenced by anything is eliminated."""
        bindings = [
            _load_const_hex("t0", "ab" * 64),
            _load_const_hex("t1", INFINITY_HEX),
            _call("t2", "ecAdd", ["t0", "t1"]),
            # t2 becomes @ref:t0, making t1 (INFINITY) unreferenced.
            # But t1 is load_const (no side effect), so it gets removed.
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        # t1 should be eliminated as dead
        names = [b.name for b in body]
        assert "t1" not in names

    def test_side_effect_bindings_preserved(self):
        """Bindings with side effects (assert, call) are never removed."""
        bindings = [
            _load_const_hex("t0", "ab" * 64),
            _load_const_hex("t1", INFINITY_HEX),
            _call("t2", "ecAdd", ["t0", "t1"]),
            _assert_ref("t3", "t2"),
        ]
        program = _make_program(bindings)
        result = optimize_ec(program)
        body = _get_method_body(result)

        # Assert binding must survive
        names = [b.name for b in body]
        assert "t3" in names


# ---------------------------------------------------------------------------
# Program structure preserved
# ---------------------------------------------------------------------------

class TestStructurePreserved:
    def test_contract_name_preserved(self):
        program = ANFProgram(
            contract_name="MyContract",
            properties=[ANFProperty(name="x", type="bigint")],
            methods=[
                ANFMethod(
                    name="doStuff",
                    params=[ANFParam(name="y", type="bigint")],
                    body=[
                        _load_const_int("t0", 1),
                        _assert_ref("t1", "t0"),
                    ],
                    is_public=True,
                ),
            ],
        )
        result = optimize_ec(program)
        assert result.contract_name == "MyContract"
        assert len(result.properties) == 1
        assert result.properties[0].name == "x"
        assert len(result.methods) == 1
        assert result.methods[0].name == "doStuff"

    def test_multiple_methods_all_optimized(self):
        """Each method is optimized independently."""
        method1 = ANFMethod(
            name="method1",
            params=[],
            body=[
                _load_const_int("t0", 0),
                _call("t1", "ecMulGen", ["t0"]),
                _assert_ref("t2", "t1"),
            ],
            is_public=True,
        )
        method2 = ANFMethod(
            name="method2",
            params=[],
            body=[
                _load_const_int("t0", 1),
                _call("t1", "ecMulGen", ["t0"]),
                _assert_ref("t2", "t1"),
            ],
            is_public=True,
        )
        program = ANFProgram(
            contract_name="Test",
            properties=[],
            methods=[method1, method2],
        )
        result = optimize_ec(program)

        # method1: ecMulGen(0) -> INFINITY
        body1 = result.methods[0].body
        t1_m1 = _find_binding(body1, "t1")
        assert t1_m1 is not None
        assert t1_m1.value.kind == "load_const"
        assert t1_m1.value.const_string == INFINITY_HEX

        # method2: ecMulGen(1) -> G
        body2 = result.methods[1].body
        t1_m2 = _find_binding(body2, "t1")
        assert t1_m2 is not None
        assert t1_m2.value.kind == "load_const"
        assert t1_m2.value.const_string == G_HEX
