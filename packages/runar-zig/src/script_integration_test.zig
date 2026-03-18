const std = @import("std");
const frontend = @import("runar_frontend");

// Compiler pipeline integration tests.
//
// These verify that .runar.zig contracts compile through the full
// Zig compiler pipeline (parse → validate → typecheck → ANF →
// stack → emit) and produce valid hex output.
//
// Script execution verification lives in bsvz, not here.

test "compileSourceToHex produces output for P2PKH" {
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

    const hex = try frontend.compileSourceToHex(allocator, source, "P2PKH.runar.zig");
    defer allocator.free(hex);

    try std.testing.expect(hex.len > 0);
    // P2PKH output should contain OP_DUP (76), OP_HASH160 (a9), OP_CHECKSIG (ac)
    try std.testing.expect(std.mem.indexOf(u8, hex, "76") != null);
    try std.testing.expect(std.mem.indexOf(u8, hex, "a9") != null);
    try std.testing.expect(std.mem.indexOf(u8, hex, "ac") != null);
}

test "compileSourceToHex produces output for arithmetic contract" {
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

    const hex = try frontend.compileSourceToHex(allocator, source, "Arithmetic.runar.zig");
    defer allocator.free(hex);

    try std.testing.expect(hex.len > 0);
    // Arithmetic output should contain OP_ADD (93), OP_SUB (94), OP_MUL (95)
    try std.testing.expect(std.mem.indexOf(u8, hex, "93") != null);
}

test "compileSourceToHex produces output for if-else contract" {
    const allocator = std.testing.allocator;

    const source =
        \\const runar = @import("runar");
        \\
        \\pub const IfElse = struct {
        \\    pub const Contract = runar.SmartContract;
        \\
        \\    limit: i64,
        \\
        \\    pub fn init(limit: i64) IfElse {
        \\        return .{ .limit = limit };
        \\    }
        \\
        \\    pub fn check(self: *const IfElse, value: i64, mode: bool) void {
        \\        var result: i64 = 0;
        \\        if (mode) {
        \\            result = value + self.limit;
        \\        } else {
        \\            result = value - self.limit;
        \\        }
        \\        runar.assert(result > 0);
        \\    }
        \\};
    ;

    const hex = try frontend.compileSourceToHex(allocator, source, "IfElse.runar.zig");
    defer allocator.free(hex);

    try std.testing.expect(hex.len > 0);
    // Should contain OP_IF (63) and OP_ELSE (67)
    try std.testing.expect(std.mem.indexOf(u8, hex, "63") != null);
    try std.testing.expect(std.mem.indexOf(u8, hex, "67") != null);
}

// Note: bounded-loop (while) is supported by the TS Zig parser but not
// the native Zig compiler parser, so we don't test it here.

test "compileSource returns full artifact JSON" {
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

    const result = try frontend.compileSource(allocator, source, "P2PKH.runar.zig");
    defer result.deinit(allocator);

    try std.testing.expect(result.script_hex.len > 0);
    try std.testing.expect(result.artifact_json != null);

    const json = result.artifact_json.?;
    // Artifact should be valid JSON with contract name and abi
    try std.testing.expect(std.mem.indexOf(u8, json, "\"contract\"") != null);
    try std.testing.expect(std.mem.indexOf(u8, json, "\"abi\"") != null);
}

test "compileSourceToHex rejects invalid source" {
    const allocator = std.testing.allocator;
    try std.testing.expectError(
        error.ParseFailed,
        frontend.compileSourceToHex(allocator, "not valid zig", "bad.runar.zig"),
    );
}
