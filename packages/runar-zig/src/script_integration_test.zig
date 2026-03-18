const std = @import("std");
const builtins = @import("builtins.zig");
const test_keys = @import("test_keys.zig");
const frontend = @import("runar_frontend");
const bsvz = @import("bsvz");

const Script = bsvz.script.Script;

// Integration tests: compile Runar contracts via the Zig compiler,
// then verify the compiled Bitcoin Script in bsvz's engine with
// real transaction context and ECDSA signatures.

fn hexToBytes(allocator: std.mem.Allocator, hex_str: []const u8) ![]u8 {
    if (hex_str.len % 2 != 0) return error.InvalidHexLength;
    const out = try allocator.alloc(u8, hex_str.len / 2);
    errdefer allocator.free(out);
    _ = try std.fmt.hexToBytes(out, hex_str);
    return out;
}

fn appendScriptNumberPush(buf: *std.ArrayListUnmanaged(u8), allocator: std.mem.Allocator, value: i64) !void {
    const encoded = try bsvz.script.ScriptNum.encode(allocator, value);
    if (encoded.len == 0) {
        try buf.append(allocator, 0x00);
    } else if (encoded.len <= 75) {
        try buf.append(allocator, @intCast(encoded.len));
        try buf.appendSlice(allocator, encoded);
    } else {
        try buf.append(allocator, 0x4c);
        try buf.append(allocator, @intCast(encoded.len));
        try buf.appendSlice(allocator, encoded);
    }
}

fn appendPushData(buf: *std.ArrayListUnmanaged(u8), allocator: std.mem.Allocator, data: []const u8) !void {
    if (data.len == 0) {
        try buf.append(allocator, 0x00);
    } else if (data.len <= 75) {
        try buf.append(allocator, @intCast(data.len));
        try buf.appendSlice(allocator, data);
    } else {
        try buf.append(allocator, 0x4c);
        try buf.append(allocator, @intCast(data.len));
        try buf.appendSlice(allocator, data);
    }
}

// ---------------------------------------------------------------------------
// Compile + verify: full pipeline
// ---------------------------------------------------------------------------

test "compile arithmetic, verify in bsvz engine (pure, no CHECKSIG)" {
    const allocator = std.testing.allocator;

    const source =
        \\const runar = @import("runar");
        \\
        \\pub const Arithmetic = struct {
        \\    pub const Contract = runar.SmartContract;
        \\
        \\    target: i64,
        \\
        \\    pub fn init(target: i64) Arithmetic {
        \\        return .{ .target = target };
        \\    }
        \\
        \\    pub fn verify(self: *const Arithmetic, a: i64, b: i64) void {
        \\        const sum = a + b;
        \\        const diff = a - b;
        \\        const prod = a * b;
        \\        const result = sum + diff + prod;
        \\        runar.assert(result == self.target);
        \\    }
        \\};
    ;

    // Compile through full Zig compiler pipeline
    const script_hex = try frontend.compileSourceToHex(allocator, source, "Arithmetic.runar.zig");
    defer allocator.free(script_hex);
    try std.testing.expect(script_hex.len > 0);

    // Decode compiled locking script
    const locking_bytes = try hexToBytes(allocator, script_hex);
    defer allocator.free(locking_bytes);
    const locking_script = Script.init(locking_bytes);

    // Build unlocking script: <a=3> <b=7> (target=27 baked into locking script via constructor)
    // a=3, b=7 → sum=10, diff=-4, prod=21 → result=27
    var unlock_buf: std.ArrayListUnmanaged(u8) = .empty;
    defer unlock_buf.deinit(allocator);
    try appendScriptNumberPush(&unlock_buf, allocator, 3);
    try appendScriptNumberPush(&unlock_buf, allocator, 7);
    const unlocking_script = Script.init(unlock_buf.items);

    // Verify through bsvz engine
    const success = try bsvz.script.engine.verifyScripts(.{
        .allocator = allocator,
    }, unlocking_script, locking_script);
    try std.testing.expect(success);
}

test "compile arithmetic, wrong args rejected by bsvz engine" {
    const allocator = std.testing.allocator;

    const source =
        \\const runar = @import("runar");
        \\
        \\pub const Arithmetic = struct {
        \\    pub const Contract = runar.SmartContract;
        \\
        \\    target: i64,
        \\
        \\    pub fn init(target: i64) Arithmetic {
        \\        return .{ .target = target };
        \\    }
        \\
        \\    pub fn verify(self: *const Arithmetic, a: i64, b: i64) void {
        \\        const sum = a + b;
        \\        const diff = a - b;
        \\        const prod = a * b;
        \\        const result = sum + diff + prod;
        \\        runar.assert(result == self.target);
        \\    }
        \\};
    ;

    const script_hex = try frontend.compileSourceToHex(allocator, source, "Arithmetic.runar.zig");
    defer allocator.free(script_hex);

    const locking_bytes = try hexToBytes(allocator, script_hex);
    defer allocator.free(locking_bytes);
    const locking_script = Script.init(locking_bytes);

    // Wrong args: a=1, b=1 → sum=2, diff=0, prod=1 → result=3, target=27 → fail
    var unlock_buf: std.ArrayListUnmanaged(u8) = .empty;
    defer unlock_buf.deinit(allocator);
    try appendScriptNumberPush(&unlock_buf, allocator, 1);
    try appendScriptNumberPush(&unlock_buf, allocator, 1);
    const unlocking_script = Script.init(unlock_buf.items);

    const success = try bsvz.script.engine.verifyScripts(.{
        .allocator = allocator,
    }, unlocking_script, locking_script);
    try std.testing.expect(!success);
}

test "compile P2PKH, verify with real ECDSA in bsvz engine" {
    const allocator = std.testing.allocator;

    const source =
        \\const runar = @import("runar");
        \\
        \\pub const P2PKH = struct {
        \\    pub const Contract = runar.SmartContract;
        \\
        \\    pubKeyHash: runar.Addr,
        \\
        \\    pub fn init(pubKeyHash: runar.Addr) P2PKH {
        \\        return .{ .pubKeyHash = pubKeyHash };
        \\    }
        \\
        \\    pub fn unlock(self: *const P2PKH, sig: runar.Sig, pubKey: runar.PubKey) void {
        \\        runar.assert(runar.hash160(pubKey) == self.pubKeyHash);
        \\        runar.assert(runar.checkSig(sig, pubKey));
        \\    }
        \\};
    ;

    const script_hex = try frontend.compileSourceToHex(allocator, source, "P2PKH.runar.zig");
    defer allocator.free(script_hex);

    const locking_bytes = try hexToBytes(allocator, script_hex);
    defer allocator.free(locking_bytes);
    const locking_script = Script.init(locking_bytes);

    // Build a real transaction for CHECKSIG
    const key_one_private = [_]u8{0} ** 31 ++ [_]u8{1};
    const private_key = try bsvz.crypto.PrivateKey.fromBytes(key_one_private);
    const previous_satoshis: i64 = 100_000;

    var inputs = [_]bsvz.transaction.Input{.{
        .previous_outpoint = .{
            .txid = .{ .bytes = [_]u8{0xaa} ** 32 },
            .index = 0,
        },
        .unlocking_script = Script.init(""),
        .sequence = 0xffff_ffff,
    }};
    var outputs = [_]bsvz.transaction.Output{.{
        .satoshis = 99_000,
        .locking_script = Script.init(&[_]u8{0x6a}),
    }};
    var tx = bsvz.transaction.Transaction{
        .version = 2,
        .inputs = &inputs,
        .outputs = &outputs,
        .lock_time = 0,
    };

    // Sign the input
    const tx_signature = try bsvz.transaction.templates.p2pkh_spend.signInput(
        allocator,
        &tx,
        0,
        locking_script,
        previous_satoshis,
        private_key,
        bsvz.transaction.templates.p2pkh_spend.default_scope,
    );
    const sig_bytes = try tx_signature.toChecksigFormat(allocator);
    defer allocator.free(sig_bytes);

    // Build unlocking script: <sig> <pubkey>
    const pubkey = try private_key.publicKey();
    var unlock_buf: std.ArrayListUnmanaged(u8) = .empty;
    defer unlock_buf.deinit(allocator);
    try appendPushData(&unlock_buf, allocator, sig_bytes);
    try appendPushData(&unlock_buf, allocator, &pubkey.bytes);
    const unlocking_script = Script.init(unlock_buf.items);

    // Verify through bsvz with full transaction context
    const success = try bsvz.script.engine.verifyScripts(.{
        .allocator = allocator,
        .tx = &tx,
        .input_index = 0,
        .previous_locking_script = locking_script,
        .previous_satoshis = previous_satoshis,
    }, unlocking_script, locking_script);
    try std.testing.expect(success);
}

test "compile P2PKH, wrong key rejected by bsvz engine" {
    const allocator = std.testing.allocator;

    const source =
        \\const runar = @import("runar");
        \\
        \\pub const P2PKH = struct {
        \\    pub const Contract = runar.SmartContract;
        \\
        \\    pubKeyHash: runar.Addr,
        \\
        \\    pub fn init(pubKeyHash: runar.Addr) P2PKH {
        \\        return .{ .pubKeyHash = pubKeyHash };
        \\    }
        \\
        \\    pub fn unlock(self: *const P2PKH, sig: runar.Sig, pubKey: runar.PubKey) void {
        \\        runar.assert(runar.hash160(pubKey) == self.pubKeyHash);
        \\        runar.assert(runar.checkSig(sig, pubKey));
        \\    }
        \\};
    ;

    const script_hex = try frontend.compileSourceToHex(allocator, source, "P2PKH.runar.zig");
    defer allocator.free(script_hex);

    const locking_bytes = try hexToBytes(allocator, script_hex);
    defer allocator.free(locking_bytes);
    const locking_script = Script.init(locking_bytes);

    // Sign with key_two but locking script has key_one's pubkeyhash
    const key_two_private = [_]u8{0} ** 31 ++ [_]u8{2};
    const wrong_key = try bsvz.crypto.PrivateKey.fromBytes(key_two_private);
    const previous_satoshis: i64 = 100_000;

    var inputs = [_]bsvz.transaction.Input{.{
        .previous_outpoint = .{ .txid = .{ .bytes = [_]u8{0xaa} ** 32 }, .index = 0 },
        .unlocking_script = Script.init(""),
        .sequence = 0xffff_ffff,
    }};
    var outputs = [_]bsvz.transaction.Output{.{
        .satoshis = 99_000,
        .locking_script = Script.init(&[_]u8{0x6a}),
    }};
    var tx = bsvz.transaction.Transaction{
        .version = 2,
        .inputs = &inputs,
        .outputs = &outputs,
        .lock_time = 0,
    };

    const tx_signature = try bsvz.transaction.templates.p2pkh_spend.signInput(
        allocator, &tx, 0, locking_script, previous_satoshis, wrong_key,
        bsvz.transaction.templates.p2pkh_spend.default_scope,
    );
    const sig_bytes = try tx_signature.toChecksigFormat(allocator);
    defer allocator.free(sig_bytes);

    const wrong_pubkey = try wrong_key.publicKey();
    var unlock_buf: std.ArrayListUnmanaged(u8) = .empty;
    defer unlock_buf.deinit(allocator);
    try appendPushData(&unlock_buf, allocator, sig_bytes);
    try appendPushData(&unlock_buf, allocator, &wrong_pubkey.bytes);
    const unlocking_script = Script.init(unlock_buf.items);

    const success = try bsvz.script.engine.verifyScripts(.{
        .allocator = allocator,
        .tx = &tx,
        .input_index = 0,
        .previous_locking_script = locking_script,
        .previous_satoshis = previous_satoshis,
    }, unlocking_script, locking_script);
    try std.testing.expect(!success);
}
