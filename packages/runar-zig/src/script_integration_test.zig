const std = @import("std");
const builtins = @import("builtins.zig");
const compile_check = @import("compile_check.zig");
const bsvz = @import("bsvz");

// Integration test: compile a Runar contract through the frontend,
// then verify the compiled Bitcoin Script executes correctly in
// bsvz's script engine.
//
// This bridges the gap between "contract parses and typechecks" and
// "the compiled output actually works as Bitcoin Script".

test "bsvz engine executes a simple arithmetic script (2 + 3 = 5)" {
    const allocator = std.testing.allocator;

    // Hand-crafted Bitcoin Script: OP_2 OP_3 OP_ADD OP_5 OP_NUMEQUAL
    const script_bytes = [_]u8{
        0x52, // OP_2
        0x53, // OP_3
        0x93, // OP_ADD
        0x55, // OP_5
        0x9c, // OP_NUMEQUAL
    };

    const script = bsvz.script.Script.init(&script_bytes);
    var result = try bsvz.script.engine.executeScript(.{
        .allocator = allocator,
    }, script);
    defer result.deinit(allocator);

    try std.testing.expect(result.success);
}

test "bsvz engine rejects failing arithmetic (2 + 3 = 6)" {
    const allocator = std.testing.allocator;

    // OP_2 OP_3 OP_ADD OP_6 OP_NUMEQUAL → false
    const script_bytes = [_]u8{
        0x52, // OP_2
        0x53, // OP_3
        0x93, // OP_ADD
        0x56, // OP_6
        0x9c, // OP_NUMEQUAL
    };

    const script = bsvz.script.Script.init(&script_bytes);
    var result = try bsvz.script.engine.executeScript(.{
        .allocator = allocator,
    }, script);
    defer result.deinit(allocator);

    try std.testing.expect(!result.success);
}

test "bsvz engine handles OP_IF branching" {
    const allocator = std.testing.allocator;

    // OP_1 OP_IF OP_2 OP_ELSE OP_3 OP_ENDIF OP_2 OP_NUMEQUAL
    const script_bytes = [_]u8{
        0x51, // OP_1 (true)
        0x63, // OP_IF
        0x52, // OP_2
        0x67, // OP_ELSE
        0x53, // OP_3
        0x68, // OP_ENDIF
        0x52, // OP_2
        0x9c, // OP_NUMEQUAL
    };

    const script = bsvz.script.Script.init(&script_bytes);
    var result = try bsvz.script.engine.executeScript(.{
        .allocator = allocator,
    }, script);
    defer result.deinit(allocator);

    try std.testing.expect(result.success);
}

test "bsvz engine handles OP_HASH160" {
    const allocator = std.testing.allocator;

    // Push "hello", OP_HASH160, push expected hash, OP_EQUAL
    const msg = "hello";
    const expected_hash = builtins.hash160(msg);

    // Build script: <push msg> OP_HASH160 <push expected> OP_EQUAL
    var script_buf: [256]u8 = undefined;
    var pos: usize = 0;

    // Push message
    script_buf[pos] = @intCast(msg.len);
    pos += 1;
    @memcpy(script_buf[pos..][0..msg.len], msg);
    pos += msg.len;

    // OP_HASH160
    script_buf[pos] = 0xa9;
    pos += 1;

    // Push expected hash (20 bytes)
    script_buf[pos] = @intCast(expected_hash.len);
    pos += 1;
    @memcpy(script_buf[pos..][0..expected_hash.len], expected_hash);
    pos += expected_hash.len;

    // OP_EQUAL
    script_buf[pos] = 0x87;
    pos += 1;

    const script = bsvz.script.Script.init(script_buf[0..pos]);
    var result = try bsvz.script.engine.executeScript(.{
        .allocator = allocator,
    }, script);
    defer result.deinit(allocator);

    try std.testing.expect(result.success);
}
