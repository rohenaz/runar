//! Pass 6: Emit -- converts Stack IR to Bitcoin Script bytes (hex string).
//!
//! Walks the StackOp list and encodes each operation as one or more Bitcoin
//! Script opcodes, producing both a hex-encoded script and a human-readable
//! ASM representation.

use serde::{Deserialize, Serialize};

use super::opcodes::opcode_byte;
use super::stack::{PushValue, StackMethod, StackOp};

// ---------------------------------------------------------------------------
// ConstructorSlot
// ---------------------------------------------------------------------------

/// Records the byte offset of a constructor parameter placeholder in the
/// emitted script. The SDK uses these offsets to splice in real values at
/// deployment time.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConstructorSlot {
    #[serde(rename = "paramIndex")]
    pub param_index: usize,
    #[serde(rename = "byteOffset")]
    pub byte_offset: usize,
}

// ---------------------------------------------------------------------------
// EmitResult
// ---------------------------------------------------------------------------

/// The output of the emission pass.
#[derive(Debug, Clone)]
pub struct EmitResult {
    pub script_hex: String,
    pub script_asm: String,
    pub constructor_slots: Vec<ConstructorSlot>,
}

// ---------------------------------------------------------------------------
// Emit context
// ---------------------------------------------------------------------------

struct EmitContext {
    hex_parts: Vec<String>,
    asm_parts: Vec<String>,
    byte_length: usize,
    constructor_slots: Vec<ConstructorSlot>,
}

impl EmitContext {
    fn new() -> Self {
        EmitContext {
            hex_parts: Vec::new(),
            asm_parts: Vec::new(),
            byte_length: 0,
            constructor_slots: Vec::new(),
        }
    }

    fn append_hex(&mut self, hex: &str) {
        self.byte_length += hex.len() / 2;
        self.hex_parts.push(hex.to_string());
    }

    fn emit_opcode(&mut self, name: &str) -> Result<(), String> {
        let byte = opcode_byte(name)
            .ok_or_else(|| format!("unknown opcode: {}", name))?;
        self.append_hex(&format!("{:02x}", byte));
        self.asm_parts.push(name.to_string());
        Ok(())
    }

    fn emit_push(&mut self, value: &PushValue) {
        let (h, a) = encode_push_value(value);
        self.append_hex(&h);
        self.asm_parts.push(a);
    }

    fn emit_placeholder(&mut self, param_index: usize, _param_name: &str) {
        let byte_offset = self.byte_length;
        self.append_hex("00"); // OP_0 placeholder byte
        self.asm_parts.push("OP_0".to_string());
        self.constructor_slots.push(ConstructorSlot {
            param_index,
            byte_offset,
        });
    }

    fn get_hex(&self) -> String {
        self.hex_parts.join("")
    }

    fn get_asm(&self) -> String {
        self.asm_parts.join(" ")
    }
}

// ---------------------------------------------------------------------------
// Script number encoding
// ---------------------------------------------------------------------------

/// Encode an i64 as a Bitcoin Script number (little-endian, sign-magnitude).
pub fn encode_script_number(n: i64) -> Vec<u8> {
    if n == 0 {
        return Vec::new();
    }

    let negative = n < 0;
    let mut abs = if negative { -(n as i128) } else { n as i128 } as u64;

    let mut bytes = Vec::new();
    while abs > 0 {
        bytes.push((abs & 0xff) as u8);
        abs >>= 8;
    }

    let last_byte = *bytes.last().unwrap();
    if last_byte & 0x80 != 0 {
        bytes.push(if negative { 0x80 } else { 0x00 });
    } else if negative {
        let len = bytes.len();
        bytes[len - 1] = last_byte | 0x80;
    }

    bytes
}

// ---------------------------------------------------------------------------
// Push data encoding
// ---------------------------------------------------------------------------

/// Encode raw bytes as a Bitcoin Script push-data operation.
pub fn encode_push_data(data: &[u8]) -> Vec<u8> {
    let len = data.len();

    if len == 0 {
        return vec![0x00]; // OP_0
    }

    if len <= 75 {
        let mut result = vec![len as u8];
        result.extend_from_slice(data);
        return result;
    }

    if len <= 255 {
        let mut result = vec![0x4c, len as u8]; // OP_PUSHDATA1
        result.extend_from_slice(data);
        return result;
    }

    if len <= 65535 {
        let mut result = vec![0x4d, (len & 0xff) as u8, ((len >> 8) & 0xff) as u8]; // OP_PUSHDATA2
        result.extend_from_slice(data);
        return result;
    }

    // OP_PUSHDATA4
    let mut result = vec![
        0x4e,
        (len & 0xff) as u8,
        ((len >> 8) & 0xff) as u8,
        ((len >> 16) & 0xff) as u8,
        ((len >> 24) & 0xff) as u8,
    ];
    result.extend_from_slice(data);
    result
}

/// Encode a push value to hex and asm strings.
fn encode_push_value(value: &PushValue) -> (String, String) {
    match value {
        PushValue::Bool(b) => {
            if *b {
                ("51".to_string(), "OP_TRUE".to_string())
            } else {
                ("00".to_string(), "OP_FALSE".to_string())
            }
        }
        PushValue::Int(n) => encode_push_int(*n),
        PushValue::Bytes(bytes) => {
            let encoded = encode_push_data(bytes);
            let h = hex::encode(&encoded);
            if bytes.is_empty() {
                (h, "OP_0".to_string())
            } else {
                (h, format!("<{}>", hex::encode(bytes)))
            }
        }
    }
}

/// Encode an integer push, using small-integer opcodes where possible.
pub fn encode_push_int(n: i64) -> (String, String) {
    if n == 0 {
        return ("00".to_string(), "OP_0".to_string());
    }

    if n == -1 {
        return ("4f".to_string(), "OP_1NEGATE".to_string());
    }

    if n >= 1 && n <= 16 {
        let opcode = 0x50 + n as u8;
        return (format!("{:02x}", opcode), format!("OP_{}", n));
    }

    let num_bytes = encode_script_number(n);
    let encoded = encode_push_data(&num_bytes);
    (hex::encode(&encoded), format!("<{}>", hex::encode(&num_bytes)))
}

// ---------------------------------------------------------------------------
// Emit a single StackOp
// ---------------------------------------------------------------------------

fn emit_stack_op(op: &StackOp, ctx: &mut EmitContext) -> Result<(), String> {
    match op {
        StackOp::Push(value) => {
            ctx.emit_push(value);
            Ok(())
        }
        StackOp::Dup => ctx.emit_opcode("OP_DUP"),
        StackOp::Swap => ctx.emit_opcode("OP_SWAP"),
        StackOp::Roll { .. } => ctx.emit_opcode("OP_ROLL"),
        StackOp::Pick { .. } => ctx.emit_opcode("OP_PICK"),
        StackOp::Drop => ctx.emit_opcode("OP_DROP"),
        StackOp::Nip => ctx.emit_opcode("OP_NIP"),
        StackOp::Over => ctx.emit_opcode("OP_OVER"),
        StackOp::Rot => ctx.emit_opcode("OP_ROT"),
        StackOp::Tuck => ctx.emit_opcode("OP_TUCK"),
        StackOp::Opcode(code) => ctx.emit_opcode(code),
        StackOp::If {
            then_ops,
            else_ops,
        } => emit_if(then_ops, else_ops, ctx),
        StackOp::Placeholder {
            param_index,
            param_name,
        } => {
            ctx.emit_placeholder(*param_index, param_name);
            Ok(())
        }
    }
}

fn emit_if(
    then_ops: &[StackOp],
    else_ops: &[StackOp],
    ctx: &mut EmitContext,
) -> Result<(), String> {
    ctx.emit_opcode("OP_IF")?;

    for op in then_ops {
        emit_stack_op(op, ctx)?;
    }

    if !else_ops.is_empty() {
        ctx.emit_opcode("OP_ELSE")?;
        for op in else_ops {
            emit_stack_op(op, ctx)?;
        }
    }

    ctx.emit_opcode("OP_ENDIF")
}

// ---------------------------------------------------------------------------
// Peephole optimization
// ---------------------------------------------------------------------------

use std::collections::HashMap;

/// Maps opcodes that can be combined with a following OP_VERIFY into a single
/// *VERIFY opcode.
fn verify_combinations() -> HashMap<&'static str, &'static str> {
    let mut m = HashMap::new();
    m.insert("OP_EQUAL", "OP_EQUALVERIFY");
    m.insert("OP_NUMEQUAL", "OP_NUMEQUALVERIFY");
    m.insert("OP_CHECKSIG", "OP_CHECKSIGVERIFY");
    m.insert("OP_CHECKMULTISIG", "OP_CHECKMULTISIGVERIFY");
    m
}

/// Peephole optimizer: combines adjacent opcode pairs into single opcodes
/// (e.g. OP_EQUAL + OP_VERIFY -> OP_EQUALVERIFY) and eliminates no-op
/// OP_SWAP OP_SWAP pairs. Recurses into If/Else blocks.
fn peephole_optimize(ops: &[StackOp]) -> Vec<StackOp> {
    let combinations = verify_combinations();
    let mut result = Vec::new();
    let mut i = 0;

    while i < ops.len() {
        let op = &ops[i];

        // Combine OP_X + OP_VERIFY -> OP_XVERIFY
        if i + 1 < ops.len() {
            let next = &ops[i + 1];
            if let (StackOp::Opcode(code), StackOp::Opcode(next_code)) = (op, next) {
                if next_code == "OP_VERIFY" {
                    if let Some(&combined) = combinations.get(code.as_str()) {
                        result.push(StackOp::Opcode(combined.to_string()));
                        i += 2; // skip the OP_VERIFY
                        continue;
                    }
                }
            }
        }

        // Eliminate OP_SWAP OP_SWAP (no-op pair)
        if i + 1 < ops.len() {
            if matches!((&ops[i], &ops[i + 1]), (StackOp::Swap, StackOp::Swap)) {
                i += 2; // skip both swaps
                continue;
            }
        }

        // Recurse into if/else blocks
        if let StackOp::If { then_ops, else_ops } = op {
            result.push(StackOp::If {
                then_ops: peephole_optimize(then_ops),
                else_ops: if else_ops.is_empty() {
                    Vec::new()
                } else {
                    peephole_optimize(else_ops)
                },
            });
            i += 1;
            continue;
        }

        result.push(op.clone());
        i += 1;
    }

    result
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/// Emit a slice of StackMethods as Bitcoin Script hex and ASM.
///
/// For contracts with multiple public methods, generates a method dispatch
/// preamble using OP_IF/OP_ELSE chains.
pub fn emit(methods: &[StackMethod]) -> Result<EmitResult, String> {
    let mut ctx = EmitContext::new();

    // Filter to public methods (exclude constructor) and apply peephole optimizations
    let public_methods: Vec<StackMethod> = methods
        .iter()
        .filter(|m| m.name != "constructor")
        .map(|m| StackMethod {
            name: m.name.clone(),
            ops: peephole_optimize(&m.ops),
            max_stack_depth: m.max_stack_depth,
        })
        .collect();

    if public_methods.is_empty() {
        return Ok(EmitResult {
            script_hex: String::new(),
            script_asm: String::new(),
            constructor_slots: Vec::new(),
        });
    }

    if public_methods.len() == 1 {
        for op in &public_methods[0].ops {
            emit_stack_op(op, &mut ctx)?;
        }
    } else {
        let refs: Vec<&StackMethod> = public_methods.iter().collect();
        emit_method_dispatch(&refs, &mut ctx)?;
    }

    Ok(EmitResult {
        script_hex: ctx.get_hex(),
        script_asm: ctx.get_asm(),
        constructor_slots: ctx.constructor_slots,
    })
}

fn emit_method_dispatch(
    methods: &[&StackMethod],
    ctx: &mut EmitContext,
) -> Result<(), String> {
    for (i, method) in methods.iter().enumerate() {
        let is_last = i == methods.len() - 1;

        if !is_last {
            ctx.emit_opcode("OP_DUP")?;
            ctx.emit_push(&PushValue::Int(i as i64));
            ctx.emit_opcode("OP_NUMEQUAL")?;
            ctx.emit_opcode("OP_IF")?;
            ctx.emit_opcode("OP_DROP")?;
        } else {
            ctx.emit_opcode("OP_DROP")?;
        }

        for op in &method.ops {
            emit_stack_op(op, ctx)?;
        }

        if !is_last {
            ctx.emit_opcode("OP_ELSE")?;
        }
    }

    // Close nested OP_IF/OP_ELSE blocks
    for _ in 0..methods.len() - 1 {
        ctx.emit_opcode("OP_ENDIF")?;
    }

    Ok(())
}

/// Emit a single method's ops. Useful for testing.
pub fn emit_method(method: &StackMethod) -> Result<EmitResult, String> {
    let mut ctx = EmitContext::new();
    for op in &method.ops {
        emit_stack_op(op, &mut ctx)?;
    }
    Ok(EmitResult {
        script_hex: ctx.get_hex(),
        script_asm: ctx.get_asm(),
        constructor_slots: ctx.constructor_slots,
    })
}
