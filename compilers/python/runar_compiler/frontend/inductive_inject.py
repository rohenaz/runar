"""Shared InductiveSmartContract internal field injection.

Used by all format-specific parsers to inject the internal fields
(_genesisOutpoint, _proof) into contracts that extend
InductiveSmartContract.
"""

from __future__ import annotations

from runar_compiler.frontend.ast_nodes import (
    PropertyNode, MethodNode, ParamNode, SourceLocation,
    PrimitiveType, Identifier, PropertyAccessExpr,
    AssignmentStmt, ExpressionStmt, CallExpr,
)

INDUCTIVE_INTERNAL_FIELDS = ("_genesisOutpoint", "_proof")


def inject_inductive_internal_props(properties: list[PropertyNode], file: str) -> None:
    """Append the internal ByteString fields for InductiveSmartContract."""
    synthetic_loc = SourceLocation(file=file, line=0, column=0)
    for name in INDUCTIVE_INTERNAL_FIELDS:
        properties.append(PropertyNode(
            name=name,
            type=PrimitiveType(name="ByteString"),
            readonly=False,
            source_location=synthetic_loc,
        ))


def inject_inductive_constructor_fields(ctor: MethodNode) -> None:
    """Inject internal field params, super() args, and assignments into the constructor."""
    bs_type = PrimitiveType(name="ByteString")

    # Add internal field params to constructor
    for name in INDUCTIVE_INTERNAL_FIELDS:
        ctor.params.append(ParamNode(name=name, type=bs_type))

    # Add internal field args to super() call (first statement)
    if ctor.body:
        first_stmt = ctor.body[0]
        if (
            isinstance(first_stmt, ExpressionStmt)
            and isinstance(first_stmt.expr, CallExpr)
            and isinstance(first_stmt.expr.callee, Identifier)
            and first_stmt.expr.callee.name == "super"
        ):
            for name in INDUCTIVE_INTERNAL_FIELDS:
                first_stmt.expr.args.append(Identifier(name=name))

    # Insert this._field = _field assignments immediately after super() call
    synthetic_loc = SourceLocation(file="", line=0, column=0)
    assignments = []
    for name in INDUCTIVE_INTERNAL_FIELDS:
        assignments.append(AssignmentStmt(
            target=PropertyAccessExpr(property=name),
            value=Identifier(name=name),
            source_location=synthetic_loc,
        ))
    # Insert after super() (index 0) so internal fields are available to developer body
    for i, a in enumerate(assignments):
        ctor.body.insert(1 + i, a)
