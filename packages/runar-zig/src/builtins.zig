const std = @import("std");
const base = @import("base.zig");
const test_keys = @import("test_keys.zig");

const Sha256Hasher = std.crypto.hash.sha2.Sha256;
const Secp256k1Ecdsa = std.crypto.sign.ecdsa.EcdsaSecp256k1Sha256;

const mock_preimage_magic = "RNRP";
const test_message = "runar-test-message-v1";
const default_zero_20 = [_]u8{0} ** 20;
const default_zero_32 = [_]u8{0} ** 32;
const default_zero_36 = [_]u8{0} ** 36;
const default_zero_64 = [_]u8{0} ** 64;

const sha256_initial_state = [_]u8{
    0x6a, 0x09, 0xe6, 0x67, 0xbb, 0x67, 0xae, 0x85,
    0x3c, 0x6e, 0xf3, 0x72, 0xa5, 0x4f, 0xf5, 0x3a,
    0x51, 0x0e, 0x52, 0x7f, 0x9b, 0x05, 0x68, 0x8c,
    0x1f, 0x83, 0xd9, 0xab, 0x5b, 0xe0, 0xcd, 0x19,
};

const sha256_k = [_]u32{
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
    0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
    0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
    0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
};

const ripemd160_r = [_]u8{
    0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
    7, 4, 13, 1, 10, 6, 15, 3, 12, 0, 9, 5, 2, 14, 11, 8,
    3, 10, 14, 4, 9, 15, 8, 1, 2, 7, 0, 6, 13, 11, 5, 12,
    1, 9, 11, 10, 0, 8, 12, 4, 13, 3, 7, 15, 14, 5, 6, 2,
    4, 0, 5, 9, 7, 12, 2, 10, 14, 1, 3, 8, 11, 6, 15, 13,
};

const ripemd160_rp = [_]u8{
    5, 14, 7, 0, 9, 2, 11, 4, 13, 6, 15, 8, 1, 10, 3, 12,
    6, 11, 3, 7, 0, 13, 5, 10, 14, 15, 8, 12, 4, 9, 1, 2,
    15, 5, 1, 3, 7, 14, 6, 9, 11, 8, 12, 2, 10, 0, 4, 13,
    8, 6, 4, 1, 3, 11, 15, 0, 5, 12, 2, 13, 9, 7, 10, 14,
    12, 15, 10, 4, 1, 5, 8, 7, 6, 2, 13, 14, 0, 3, 9, 11,
};

const ripemd160_s = [_]u5{
    11, 14, 15, 12, 5, 8, 7, 9, 11, 13, 14, 15, 6, 7, 9, 8,
    7, 6, 8, 13, 11, 9, 7, 15, 7, 12, 15, 9, 11, 7, 13, 12,
    11, 13, 6, 7, 14, 9, 13, 15, 14, 8, 13, 6, 5, 12, 7, 5,
    11, 12, 14, 15, 14, 15, 9, 8, 9, 14, 5, 6, 8, 6, 5, 12,
    9, 15, 5, 11, 6, 8, 13, 12, 5, 12, 13, 14, 11, 8, 5, 6,
};

const ripemd160_sp = [_]u5{
    8, 9, 9, 11, 13, 15, 15, 5, 7, 7, 8, 11, 14, 14, 12, 6,
    9, 13, 15, 7, 12, 8, 9, 11, 7, 7, 12, 7, 6, 15, 13, 11,
    9, 7, 15, 11, 8, 6, 6, 14, 12, 13, 5, 14, 13, 13, 7, 5,
    15, 5, 8, 11, 14, 14, 6, 14, 6, 9, 12, 9, 12, 5, 15, 8,
    8, 5, 12, 9, 12, 5, 14, 6, 8, 13, 6, 5, 15, 13, 11, 11,
};

const ripemd160_k = [_]u32{
    0x00000000,
    0x5a827999,
    0x6ed9eba1,
    0x8f1bbcdc,
    0xa953fd4e,
};

const ripemd160_kp = [_]u32{
    0x50a28be6,
    0x5c4dd124,
    0x6d703ef3,
    0x7a6d76e9,
    0x00000000,
};

const blake3_iv_words = [_]u32{
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
    0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
};

const blake3_iv_bytes = [_]u8{
    0x6a, 0x09, 0xe6, 0x67, 0xbb, 0x67, 0xae, 0x85,
    0x3c, 0x6e, 0xf3, 0x72, 0xa5, 0x4f, 0xf5, 0x3a,
    0x51, 0x0e, 0x52, 0x7f, 0x9b, 0x05, 0x68, 0x8c,
    0x1f, 0x83, 0xd9, 0xab, 0x5b, 0xe0, 0xcd, 0x19,
};

const blake3_msg_perm = [_]u8{ 2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8 };
const wots_w = 16;
const wots_n = 32;
const wots_len1 = 64;
const wots_len2 = 3;
const wots_len = wots_len1 + wots_len2;

pub const MockPreimageParts = struct {
    hashPrevouts: base.Sha256 = default_zero_32[0..],
    outpoint: base.ByteString = default_zero_36[0..],
    outputHash: base.Sha256 = default_zero_32[0..],
    locktime: base.Bigint = 0,
};

pub fn assert(condition: bool) void {
    if (!condition) @panic("runar assertion failed");
}

pub fn sha256(data: base.ByteString) base.Sha256 {
    var out: [32]u8 = undefined;
    Sha256Hasher.hash(data, &out, .{});
    return dupeBytes(&out);
}

pub fn ripemd160(data: base.ByteString) base.Ripemd160 {
    var out: [20]u8 = undefined;
    ripemd160Hash(&out, data);
    return dupeBytes(&out);
}

pub fn hash160(data: base.ByteString) base.Addr {
    const first = sha256(data);
    defer freeIfOwned(first);
    return ripemd160(first);
}

pub fn hash256(data: base.ByteString) base.Sha256 {
    const first = sha256(data);
    defer freeIfOwned(first);
    return sha256(first);
}

pub fn bytesEq(left: base.ByteString, right: base.ByteString) bool {
    return std.mem.eql(u8, left, right);
}

pub fn checkSig(sig: base.Sig, pub_key: base.PubKey) bool {
    if (sig.len < 8 or pub_key.len == 0) return false;

    const public_key = Secp256k1Ecdsa.PublicKey.fromSec1(pub_key) catch return false;
    const der_sig = stripSigHashByte(sig);
    const parsed_sig = Secp256k1Ecdsa.Signature.fromDer(der_sig) catch return false;

    parsed_sig.verify(test_message, public_key) catch return false;
    return true;
}

pub fn checkMultiSig(sigs: []const base.Sig, pub_keys: []const base.PubKey) bool {
    if (sigs.len > pub_keys.len) return false;

    var pub_key_index: usize = 0;
    for (sigs) |sig| {
        var matched = false;
        while (pub_key_index < pub_keys.len) {
            if (checkSig(sig, pub_keys[pub_key_index])) {
                pub_key_index += 1;
                matched = true;
                break;
            }
            pub_key_index += 1;
        }
        if (!matched) return false;
    }
    return true;
}

pub fn checkPreimage(preimage: base.SigHashPreimage) bool {
    return preimage.len >= 4 and std.mem.eql(u8, preimage[0..4], mock_preimage_magic);
}

pub fn signTestMessage(pair: test_keys.TestKeyPair) base.Sig {
    const secret_key = parseFixtureSecretKey(pair.privKey) catch @panic("invalid fixture private key");
    const key_pair = Secp256k1Ecdsa.KeyPair.fromSecretKey(secret_key) catch @panic("invalid fixture private key");
    const derived_pub_key = key_pair.public_key.toCompressedSec1();
    if (!std.mem.eql(u8, &derived_pub_key, pair.pubKey)) {
        @panic("fixture private/public key mismatch");
    }

    const sig = key_pair.sign(test_message, null) catch @panic("unable to sign fixture test message");
    var der_buf: [Secp256k1Ecdsa.Signature.der_encoded_length_max]u8 = undefined;
    return dupeBytes(sig.toDer(&der_buf));
}

pub fn mockPreimage(parts: MockPreimageParts) base.SigHashPreimage {
    var encoded = std.heap.page_allocator.alloc(u8, 4 + 32 + 36 + 32 + 8) catch @panic("OOM");
    @memcpy(encoded[0..4], mock_preimage_magic);
    copyFixed(encoded[4..36], parts.hashPrevouts);
    copyFixed(encoded[36..72], parts.outpoint);
    copyFixed(encoded[72..104], parts.outputHash);
    encodeInt64Le(encoded[104..112], parts.locktime);
    return encoded;
}

pub fn extractHashPrevouts(preimage: base.SigHashPreimage) base.Sha256 {
    return sliceOrZero(preimage, 4, 32);
}

pub fn extractOutpoint(preimage: base.SigHashPreimage) base.ByteString {
    return sliceOrZero(preimage, 36, 36);
}

pub fn extractOutputHash(preimage: base.SigHashPreimage) base.Sha256 {
    return sliceOrZero(preimage, 72, 32);
}

pub fn extractLocktime(preimage: base.SigHashPreimage) base.Bigint {
    if (preimage.len < 112) return 0;
    return decodeInt64Le(preimage[104..112]);
}

pub fn cat(left: base.ByteString, right: base.ByteString) base.ByteString {
    var out = std.heap.page_allocator.alloc(u8, left.len + right.len) catch @panic("OOM");
    @memcpy(out[0..left.len], left);
    @memcpy(out[left.len..], right);
    return out;
}

pub fn substr(bytes: base.ByteString, start: base.Bigint, len: base.Bigint) base.ByteString {
    if (start < 0 or len <= 0) return &.{};
    const start_usize = std.math.cast(usize, start) orelse return &.{};
    const len_usize = std.math.cast(usize, len) orelse return &.{};
    if (start_usize >= bytes.len) return &.{};

    const remaining = bytes.len - start_usize;
    const end_usize = start_usize + @min(len_usize, remaining);
    return dupeBytes(bytes[start_usize..end_usize]);
}

pub fn num2bin(value: base.Bigint, size: base.Bigint) base.ByteString {
    if (size < 0) return &.{};
    const size_usize = std.math.cast(usize, size) orelse return &.{};
    if (size_usize == 0) {
        if (value == 0) return &.{};
        @panic("num2bin: size too small");
    }

    var out = std.heap.page_allocator.alloc(u8, size_usize) catch @panic("OOM");
    @memset(out, 0);

    if (value == 0) return out;

    var magnitude = unsignedAbs(value);
    var encoded: [9]u8 = undefined;
    var encoded_len: usize = 0;
    while (magnitude != 0) : (encoded_len += 1) {
        encoded[encoded_len] = @truncate(magnitude & 0xff);
        magnitude >>= 8;
    }

    if ((encoded[encoded_len - 1] & 0x80) != 0) {
        encoded[encoded_len] = 0;
        encoded_len += 1;
    }

    if (encoded_len > size_usize) @panic("num2bin: size too small");

    @memcpy(out[0..encoded_len], encoded[0..encoded_len]);
    if (value < 0) out[size_usize - 1] |= 0x80;
    return out;
}

pub fn bin2num(bytes: base.ByteString) base.Bigint {
    if (bytes.len == 0) return 0;

    const last_index = bytes.len - 1;
    const negative = (bytes[last_index] & 0x80) != 0;

    var magnitude: u64 = 0;
    for (bytes, 0..) |byte, index| {
        const part: u8 = if (index == last_index) (byte & 0x7f) else byte;
        if (index >= 8) {
            if (part != 0) @panic("bin2num: magnitude too large");
            continue;
        }
        magnitude |= @as(u64, part) << @intCast(index * 8);
    }

    if (!negative) {
        if (magnitude > std.math.maxInt(i64)) @panic("bin2num: magnitude too large");
        return @intCast(magnitude);
    }

    if (magnitude == 0) return 0;
    if (magnitude == (@as(u64, 1) << 63)) return std.math.minInt(i64);
    if (magnitude > std.math.maxInt(i64)) @panic("bin2num: magnitude too large");
    return -@as(i64, @intCast(magnitude));
}

pub fn clamp(value: base.Bigint, lo: base.Bigint, hi: base.Bigint) base.Bigint {
    return @max(lo, @min(hi, value));
}

pub fn safediv(lhs: base.Bigint, rhs: base.Bigint) base.Bigint {
    if (rhs == 0) return 0;
    return @divTrunc(lhs, rhs);
}

pub fn safemod(lhs: base.Bigint, rhs: base.Bigint) base.Bigint {
    if (rhs == 0) return 0;
    return @rem(lhs, rhs);
}

pub fn sign(value: base.Bigint) base.Bigint {
    return if (value < 0) -1 else if (value > 0) 1 else 0;
}

pub fn pow(base_value: base.Bigint, exponent: base.Bigint) base.Bigint {
    if (exponent < 0) @panic("pow: negative exponent");
    if (exponent == 0) return 1;

    var result: i64 = 1;
    var factor = base_value;
    var remaining: u64 = @intCast(exponent);
    while (remaining != 0) : (remaining >>= 1) {
        if ((remaining & 1) != 0) result = checkedMul(result, factor);
        if (remaining > 1) factor = checkedMul(factor, factor);
    }
    return result;
}

pub fn mulDiv(a: base.Bigint, b: base.Bigint, divisor: base.Bigint) base.Bigint {
    if (divisor == 0) return 0;
    return @divTrunc(checkedMul(a, b), divisor);
}

pub fn percentOf(value: base.Bigint, percentage: base.Bigint) base.Bigint {
    return @divTrunc(checkedMul(value, percentage), 100);
}

pub fn sqrt(value: base.Bigint) base.Bigint {
    if (value <= 0) return 0;

    var x = value;
    var y = @divTrunc(value, 2) + 1;
    while (y < x) {
        x = y;
        y = @divTrunc(y + @divTrunc(value, y), 2);
    }
    return x;
}

pub fn gcd(a: base.Bigint, b: base.Bigint) base.Bigint {
    var x = checkedAbs(a);
    var y = checkedAbs(b);
    while (y != 0) {
        const next = @mod(x, y);
        x = y;
        y = next;
    }
    return x;
}

pub fn log2(value: base.Bigint) base.Bigint {
    if (value <= 1) return 0;

    var count: i64 = 0;
    var current = value;
    while (current > 1) : (count += 1) {
        current = @divTrunc(current, 2);
    }
    return count;
}

pub fn sha256Compress(chaining_value: base.ByteString, block: base.ByteString) base.ByteString {
    if (chaining_value.len != 32) @panic("sha256Compress: state must be 32 bytes");
    if (block.len != 64) @panic("sha256Compress: block must be 64 bytes");

    var out: [32]u8 = undefined;
    sha256CompressBlock(&out, chaining_value, block);
    return dupeBytes(&out);
}

pub fn sha256Finalize(chaining_value: base.ByteString, remaining: base.ByteString, total_len: base.Bigint) base.ByteString {
    if (chaining_value.len != 32) @panic("sha256Finalize: state must be 32 bytes");
    if (remaining.len > 119) @panic("sha256Finalize: remaining must be <= 119 bytes");
    if (total_len < 0) @panic("sha256Finalize: total bit length must be non-negative");

    const blocks: usize = if (remaining.len + 1 + 8 <= 64) 1 else 2;
    const total_bytes = blocks * 64;

    var padded = [_]u8{0} ** 128;
    @memcpy(padded[0..remaining.len], remaining);
    padded[remaining.len] = 0x80;
    std.mem.writeInt(u64, padded[total_bytes - 8 .. total_bytes][0..8], @intCast(total_len), .big);

    var out: [32]u8 = undefined;
    if (blocks == 1) {
        sha256CompressBlock(&out, chaining_value, padded[0..64]);
        return dupeBytes(&out);
    }

    var mid: [32]u8 = undefined;
    sha256CompressBlock(&mid, chaining_value, padded[0..64]);
    sha256CompressBlock(&out, &mid, padded[64..128]);
    return dupeBytes(&out);
}

pub fn blake3Compress(chaining_value: base.ByteString, block: base.ByteString) base.ByteString {
    if (chaining_value.len != 32) @panic("blake3Compress: chaining value must be 32 bytes");
    if (block.len != 64) @panic("blake3Compress: block must be 64 bytes");

    var h: [8]u32 = undefined;
    var m: [16]u32 = undefined;
    for (0..8) |index| {
        h[index] = std.mem.readInt(u32, chaining_value[index * 4 ..][0..4], .big);
    }
    for (0..16) |index| {
        m[index] = std.mem.readInt(u32, block[index * 4 ..][0..4], .big);
    }

    var state = [_]u32{
        h[0], h[1], h[2], h[3],
        h[4], h[5], h[6], h[7],
        blake3_iv_words[0], blake3_iv_words[1], blake3_iv_words[2], blake3_iv_words[3],
        0, 0, 64, 11,
    };
    var msg = m;
    for (0..7) |round_index| {
        blake3Round(&state, &msg);
        if (round_index < 6) msg = blake3Permute(msg);
    }

    var out: [32]u8 = undefined;
    for (0..8) |index| {
        const word = state[index] ^ state[index + 8];
        std.mem.writeInt(u32, out[index * 4 ..][0..4], word, .big);
    }
    return dupeBytes(&out);
}

pub fn blake3Hash(message: base.ByteString) base.ByteString {
    if (message.len > 64) @panic("blake3Hash: message must be <= 64 bytes");

    var block = [_]u8{0} ** 64;
    @memcpy(block[0..message.len], message);
    return blake3Compress(blake3_iv_bytes[0..], &block);
}

pub fn verifyRabinSig(message: base.ByteString, sig: base.RabinSig, padding: base.ByteString, pub_key: base.RabinPubKey) bool {
    var modulus = BigUint.fromLeBytes(std.heap.page_allocator, pub_key) catch return false;
    defer modulus.deinit();
    if (modulus.isZero()) return false;

    var hash_bytes: [32]u8 = undefined;
    Sha256Hasher.hash(message, &hash_bytes, .{});

    var hash_bn = BigUint.fromLeBytes(std.heap.page_allocator, &hash_bytes) catch return false;
    defer hash_bn.deinit();
    var sig_bn = BigUint.fromLeBytes(std.heap.page_allocator, sig) catch return false;
    defer sig_bn.deinit();
    var pad_bn = BigUint.fromLeBytes(std.heap.page_allocator, padding) catch return false;
    defer pad_bn.deinit();

    var sig_sq = sig_bn.mul(&sig_bn) catch return false;
    defer sig_sq.deinit();
    var lhs_sum = sig_sq.add(&pad_bn) catch return false;
    defer lhs_sum.deinit();
    var lhs = lhs_sum.rem(&modulus) catch return false;
    defer lhs.deinit();
    var rhs = hash_bn.rem(&modulus) catch return false;
    defer rhs.deinit();

    return lhs.eql(&rhs);
}

pub fn verifyWOTS(message: base.ByteString, sig: base.ByteString, pub_key: base.ByteString) bool {
    if (sig.len != wots_len * wots_n) return false;
    if (pub_key.len != 2 * wots_n) return false;

    const pub_seed = pub_key[0..wots_n];
    const pk_root = pub_key[wots_n .. 2 * wots_n];
    var msg_hash: [32]u8 = undefined;
    Sha256Hasher.hash(message, &msg_hash, .{});

    const digits = wotsAllDigits(&msg_hash);
    var endpoints = std.heap.page_allocator.alloc(u8, wots_len * wots_n) catch @panic("OOM");
    defer std.heap.page_allocator.free(endpoints);

    for (0..wots_len) |i| {
        const sig_element = sig[i * wots_n ..][0..wots_n];
        const remaining = (wots_w - 1) - digits[i];
        const endpoint = wotsChain(sig_element, digits[i], remaining, pub_seed, i);
        @memcpy(endpoints[i * wots_n ..][0..wots_n], &endpoint);
    }

    var computed_root: [32]u8 = undefined;
    Sha256Hasher.hash(endpoints, &computed_root, .{});
    return std.mem.eql(u8, &computed_root, pk_root);
}

pub fn verifySLHDSA_SHA2_128s(message: base.ByteString, sig: base.ByteString, pub_key: base.ByteString) bool {
    _ = message;
    _ = sig;
    _ = pub_key;
    return false;
}

pub fn verifySLHDSA_SHA2_128f(message: base.ByteString, sig: base.ByteString, pub_key: base.ByteString) bool {
    _ = message;
    _ = sig;
    _ = pub_key;
    return false;
}

pub fn verifySLHDSA_SHA2_192s(message: base.ByteString, sig: base.ByteString, pub_key: base.ByteString) bool {
    _ = message;
    _ = sig;
    _ = pub_key;
    return false;
}

pub fn verifySLHDSA_SHA2_192f(message: base.ByteString, sig: base.ByteString, pub_key: base.ByteString) bool {
    _ = message;
    _ = sig;
    _ = pub_key;
    return false;
}

pub fn verifySLHDSA_SHA2_256s(message: base.ByteString, sig: base.ByteString, pub_key: base.ByteString) bool {
    _ = message;
    _ = sig;
    _ = pub_key;
    return false;
}

pub fn verifySLHDSA_SHA2_256f(message: base.ByteString, sig: base.ByteString, pub_key: base.ByteString) bool {
    _ = message;
    _ = sig;
    _ = pub_key;
    return false;
}

pub fn ecMakePoint(x: base.Bigint, y: base.Bigint) base.Point {
    var point = [_]u8{0} ** 64;
    std.mem.writeInt(u64, point[24..32], @bitCast(x), .big);
    std.mem.writeInt(u64, point[56..64], @bitCast(y), .big);
    return dupeBytes(&point);
}

pub fn ecPointX(point: base.Point) base.Bigint {
    if (point.len != 64) @panic("ecPointX: point must be 64 bytes");
    return @bitCast(std.mem.readInt(u64, point[24..32], .big));
}

pub fn ecPointY(point: base.Point) base.Bigint {
    if (point.len != 64) @panic("ecPointY: point must be 64 bytes");
    return @bitCast(std.mem.readInt(u64, point[56..64], .big));
}

pub fn ecAdd(left: base.Point, right: base.Point) base.Point {
    const lp = parsePoint(left) catch @panic("ecAdd: invalid point");
    const rp = parsePoint(right) catch @panic("ecAdd: invalid point");
    return serializePoint(lp.add(rp));
}

pub fn ecMul(point: base.Point, scalar: base.Bigint) base.Point {
    const p = parsePoint(point) catch @panic("ecMul: invalid point");
    if (scalar == 0) return dupeBytes(&([_]u8{0} ** 64));
    if (isIdentityPoint(point)) return dupeBytes(&([_]u8{0} ** 64));

    const abs_scalar = scalarBytesFromI64(scalar);
    var result = p.mul(abs_scalar, .big) catch @panic("ecMul: invalid scalar");
    if (scalar < 0) result = result.neg();
    return serializePoint(result);
}

pub fn ecMulGen(scalar: base.Bigint) base.Point {
    if (scalar == 0) return dupeBytes(&([_]u8{0} ** 64));

    const abs_scalar = scalarBytesFromI64(scalar);
    var result = std.crypto.ecc.Secp256k1.basePoint.mul(abs_scalar, .big) catch @panic("ecMulGen: invalid scalar");
    if (scalar < 0) result = result.neg();
    return serializePoint(result);
}

pub fn ecNegate(point: base.Point) base.Point {
    const p = parsePoint(point) catch @panic("ecNegate: invalid point");
    return serializePoint(p.neg());
}

pub fn ecOnCurve(point: base.Point) bool {
    _ = parsePoint(point) catch return false;
    return true;
}

pub fn ecModReduce(value: base.Bigint, modulus: base.Bigint) base.Bigint {
    if (modulus == 0) return 0;
    const reduced = @mod(value, modulus);
    return if (reduced < 0) reduced + modulus else reduced;
}

pub fn ecEncodeCompressed(point: base.Point) base.ByteString {
    const p = parsePoint(point) catch @panic("ecEncodeCompressed: invalid point");
    if (isIdentityPoint(point)) return dupeBytes(&[_]u8{0x00});
    const compressed = p.toCompressedSec1();
    return dupeBytes(&compressed);
}

fn parseFixtureSecretKey(priv_key_hex: []const u8) !Secp256k1Ecdsa.SecretKey {
    var secret_key_bytes: [Secp256k1Ecdsa.SecretKey.encoded_length]u8 = undefined;
    _ = try std.fmt.hexToBytes(&secret_key_bytes, priv_key_hex);
    return Secp256k1Ecdsa.SecretKey.fromBytes(secret_key_bytes);
}

fn stripSigHashByte(sig: []const u8) []const u8 {
    if (sig.len < 2 or sig[0] != 0x30) return sig;

    const pure_der_len = @as(usize, sig[1]) + 2;
    if (sig.len == pure_der_len + 1) return sig[0..pure_der_len];
    return sig;
}

fn dupeBytes(bytes: []const u8) []const u8 {
    return std.heap.page_allocator.dupe(u8, bytes) catch @panic("OOM");
}

fn freeIfOwned(bytes: []const u8) void {
    if (bytes.len == 0) return;
    const addr = @intFromPtr(bytes.ptr);
    const static_addrs = [_]usize{
        @intFromPtr(default_zero_20[0..].ptr),
        @intFromPtr(default_zero_32[0..].ptr),
        @intFromPtr(default_zero_36[0..].ptr),
        @intFromPtr(default_zero_64[0..].ptr),
        @intFromPtr(mock_preimage_magic.ptr),
        @intFromPtr(sha256_initial_state[0..].ptr),
        @intFromPtr(blake3_iv_bytes[0..].ptr),
    };
    for (static_addrs) |static_addr| {
        if (addr == static_addr) return;
    }
    std.heap.page_allocator.free(bytes);
}

fn copyFixed(dest: []u8, source: []const u8) void {
    const count = @min(dest.len, source.len);
    @memset(dest, 0);
    @memcpy(dest[0..count], source[0..count]);
}

fn sliceOrZero(bytes: []const u8, start: usize, len: usize) []const u8 {
    if (start > bytes.len or len > bytes.len - start) {
        const zeros = std.heap.page_allocator.alloc(u8, len) catch @panic("OOM");
        @memset(zeros, 0);
        return zeros;
    }
    return dupeBytes(bytes[start .. start + len]);
}

fn encodeInt64Le(dest: []u8, value: i64) void {
    const tmp = @as(u64, @bitCast(value));
    for (dest, 0..) |*byte, index| {
        byte.* = @truncate(tmp >> @intCast(index * 8));
    }
}

fn decodeInt64Le(bytes: []const u8) i64 {
    var value: u64 = 0;
    for (bytes, 0..) |byte, index| {
        value |= @as(u64, byte) << @intCast(index * 8);
    }
    return @bitCast(value);
}

fn checkedMul(lhs: i64, rhs: i64) i64 {
    const result = @mulWithOverflow(lhs, rhs);
    if (result[1] != 0) @panic("runar integer overflow");
    return result[0];
}

fn checkedAbs(value: i64) i64 {
    if (value == std.math.minInt(i64)) @panic("runar integer overflow");
    return if (value < 0) -value else value;
}

fn unsignedAbs(value: i64) u64 {
    if (value >= 0) return @intCast(value);
    if (value == std.math.minInt(i64)) return @as(u64, 1) << 63;
    return @intCast(-value);
}

fn sha256CompressBlock(out: *[32]u8, state_bytes: []const u8, block_bytes: []const u8) void {
    var h: [8]u32 = undefined;
    var w: [64]u32 = undefined;

    for (0..8) |index| {
        h[index] = std.mem.readInt(u32, state_bytes[index * 4 ..][0..4], .big);
    }
    for (0..16) |index| {
        w[index] = std.mem.readInt(u32, block_bytes[index * 4 ..][0..4], .big);
    }
    for (16..64) |index| {
        const s0 = std.math.rotr(u32, w[index - 15], 7) ^ std.math.rotr(u32, w[index - 15], 18) ^ (w[index - 15] >> 3);
        const s1 = std.math.rotr(u32, w[index - 2], 17) ^ std.math.rotr(u32, w[index - 2], 19) ^ (w[index - 2] >> 10);
        w[index] = w[index - 16] +% s0 +% w[index - 7] +% s1;
    }

    var a = h[0];
    var b = h[1];
    var c = h[2];
    var d = h[3];
    var e = h[4];
    var f = h[5];
    var g = h[6];
    var hh = h[7];

    for (0..64) |index| {
        const big_s1 = std.math.rotr(u32, e, 6) ^ std.math.rotr(u32, e, 11) ^ std.math.rotr(u32, e, 25);
        const ch = (e & f) ^ ((~e) & g);
        const temp1 = hh +% big_s1 +% ch +% sha256_k[index] +% w[index];
        const big_s0 = std.math.rotr(u32, a, 2) ^ std.math.rotr(u32, a, 13) ^ std.math.rotr(u32, a, 22);
        const maj = (a & b) ^ (a & c) ^ (b & c);
        const temp2 = big_s0 +% maj;

        hh = g;
        g = f;
        f = e;
        e = d +% temp1;
        d = c;
        c = b;
        b = a;
        a = temp1 +% temp2;
    }

    h[0] +%= a;
    h[1] +%= b;
    h[2] +%= c;
    h[3] +%= d;
    h[4] +%= e;
    h[5] +%= f;
    h[6] +%= g;
    h[7] +%= hh;

    for (0..8) |index| {
        std.mem.writeInt(u32, out[index * 4 ..][0..4], h[index], .big);
    }
}

fn ripemd160Hash(out: *[20]u8, data: []const u8) void {
    const total_len = ((data.len + 9 + 63) / 64) * 64;
    var padded = std.heap.page_allocator.alloc(u8, total_len) catch @panic("OOM");
    defer std.heap.page_allocator.free(padded);

    @memset(padded, 0);
    @memcpy(padded[0..data.len], data);
    padded[data.len] = 0x80;
    std.mem.writeInt(u64, padded[total_len - 8 .. total_len][0..8], @as(u64, @intCast(data.len)) * 8, .little);

    var h0: u32 = 0x67452301;
    var h1: u32 = 0xefcdab89;
    var h2: u32 = 0x98badcfe;
    var h3: u32 = 0x10325476;
    var h4: u32 = 0xc3d2e1f0;

    for (0..total_len / 64) |block_index| {
        const block = padded[block_index * 64 ..][0..64];
        var x: [16]u32 = undefined;
        for (0..16) |word_index| {
            x[word_index] = std.mem.readInt(u32, block[word_index * 4 ..][0..4], .little);
        }

        var al = h0;
        var bl = h1;
        var cl = h2;
        var dl = h3;
        var el = h4;
        var ar = h0;
        var br = h1;
        var cr = h2;
        var dr = h3;
        var er = h4;

        for (0..80) |step| {
            const round = step / 16;

            const tl = std.math.rotl(
                u32,
                al +% ripemd160F(step, bl, cl, dl) +% x[ripemd160_r[step]] +% ripemd160_k[round],
                ripemd160_s[step],
            ) +% el;
            al = el;
            el = dl;
            dl = std.math.rotl(u32, cl, 10);
            cl = bl;
            bl = tl;

            const tr = std.math.rotl(
                u32,
                ar +% ripemd160F(79 - step, br, cr, dr) +% x[ripemd160_rp[step]] +% ripemd160_kp[round],
                ripemd160_sp[step],
            ) +% er;
            ar = er;
            er = dr;
            dr = std.math.rotl(u32, cr, 10);
            cr = br;
            br = tr;
        }

        const t = h1 +% cl +% dr;
        h1 = h2 +% dl +% er;
        h2 = h3 +% el +% ar;
        h3 = h4 +% al +% br;
        h4 = h0 +% bl +% cr;
        h0 = t;
    }

    std.mem.writeInt(u32, out[0..4], h0, .little);
    std.mem.writeInt(u32, out[4..8], h1, .little);
    std.mem.writeInt(u32, out[8..12], h2, .little);
    std.mem.writeInt(u32, out[12..16], h3, .little);
    std.mem.writeInt(u32, out[16..20], h4, .little);
}

fn ripemd160F(step: usize, x: u32, y: u32, z: u32) u32 {
    return switch (step / 16) {
        0 => x ^ y ^ z,
        1 => (x & y) | (~x & z),
        2 => (x | ~y) ^ z,
        3 => (x & z) | (y & ~z),
        else => x ^ (y | ~z),
    };
}

fn blake3Round(state: *[16]u32, msg: *const [16]u32) void {
    blake3G(state, 0, 4, 8, 12, msg[0], msg[1]);
    blake3G(state, 1, 5, 9, 13, msg[2], msg[3]);
    blake3G(state, 2, 6, 10, 14, msg[4], msg[5]);
    blake3G(state, 3, 7, 11, 15, msg[6], msg[7]);
    blake3G(state, 0, 5, 10, 15, msg[8], msg[9]);
    blake3G(state, 1, 6, 11, 12, msg[10], msg[11]);
    blake3G(state, 2, 7, 8, 13, msg[12], msg[13]);
    blake3G(state, 3, 4, 9, 14, msg[14], msg[15]);
}

fn blake3G(state: *[16]u32, a: usize, b: usize, c: usize, d: usize, mx: u32, my: u32) void {
    state[a] = state[a] +% state[b] +% mx;
    state[d] = std.math.rotr(u32, state[d] ^ state[a], 16);
    state[c] = state[c] +% state[d];
    state[b] = std.math.rotr(u32, state[b] ^ state[c], 12);
    state[a] = state[a] +% state[b] +% my;
    state[d] = std.math.rotr(u32, state[d] ^ state[a], 8);
    state[c] = state[c] +% state[d];
    state[b] = std.math.rotr(u32, state[b] ^ state[c], 7);
}

fn blake3Permute(msg: [16]u32) [16]u32 {
    var out: [16]u32 = undefined;
    for (0..16) |index| {
        out[index] = msg[blake3_msg_perm[index]];
    }
    return out;
}

const BigUint = struct {
    allocator: std.mem.Allocator,
    limbs: []u64,

    fn zero(allocator: std.mem.Allocator) !BigUint {
        var limbs = try allocator.alloc(u64, 1);
        limbs[0] = 0;
        return .{ .allocator = allocator, .limbs = limbs };
    }

    fn fromLeBytes(allocator: std.mem.Allocator, bytes: []const u8) !BigUint {
        if (bytes.len == 0) return zero(allocator);

        const limb_count = std.math.divCeil(usize, bytes.len, 8) catch unreachable;
        var limbs = try allocator.alloc(u64, limb_count);
        @memset(limbs, 0);
        var offset: usize = 0;
        while (offset < bytes.len) : (offset += 8) {
            var limb: u64 = 0;
            for (0..8) |j| {
                if (offset + j < bytes.len) {
                    limb |= @as(u64, bytes[offset + j]) << @intCast(j * 8);
                }
            }
            limbs[offset / 8] = limb;
        }
        return normalizeOwned(allocator, limbs);
    }

    fn deinit(self: *BigUint) void {
        self.allocator.free(self.limbs);
        self.* = undefined;
    }

    fn isZero(self: *const BigUint) bool {
        for (self.limbs) |limb| {
            if (limb != 0) return false;
        }
        return true;
    }

    fn eql(self: *const BigUint, other: *const BigUint) bool {
        if (self.limbs.len != other.limbs.len) return false;
        return std.mem.eql(u64, self.limbs, other.limbs);
    }

    fn cmp(self: *const BigUint, other: *const BigUint) std.math.Order {
        if (self.limbs.len != other.limbs.len) return std.math.order(self.limbs.len, other.limbs.len);
        var i = self.limbs.len;
        while (i != 0) {
            i -= 1;
            if (self.limbs[i] != other.limbs[i]) return std.math.order(self.limbs[i], other.limbs[i]);
        }
        return .eq;
    }

    fn add(self: *const BigUint, other: *const BigUint) !BigUint {
        const max_len = @max(self.limbs.len, other.limbs.len);
        var result = try self.allocator.alloc(u64, max_len + 1);
        var carry: u64 = 0;
        for (0..max_len) |i| {
            const a = if (i < self.limbs.len) self.limbs[i] else 0;
            const b = if (i < other.limbs.len) other.limbs[i] else 0;
            const sum1 = @addWithOverflow(a, b);
            const sum2 = @addWithOverflow(sum1[0], carry);
            result[i] = sum2[0];
            carry = sum1[1] + sum2[1];
        }
        result[max_len] = carry;
        return normalizeOwned(self.allocator, result);
    }

    fn sub(self: *const BigUint, other: *const BigUint) !BigUint {
        if (self.cmp(other) == .lt) return error.BigUintUnderflow;

        var result = try self.allocator.alloc(u64, self.limbs.len);
        var borrow: u64 = 0;
        for (0..self.limbs.len) |i| {
            const a = self.limbs[i];
            const b = if (i < other.limbs.len) other.limbs[i] else 0;
            const sub1 = @subWithOverflow(a, b);
            const sub2 = @subWithOverflow(sub1[0], borrow);
            result[i] = sub2[0];
            borrow = sub1[1] + sub2[1];
        }
        return normalizeOwned(self.allocator, result);
    }

    fn mul(self: *const BigUint, other: *const BigUint) !BigUint {
        var result = try self.allocator.alloc(u64, self.limbs.len + other.limbs.len);
        @memset(result, 0);

        for (0..self.limbs.len) |i| {
            var carry: u64 = 0;
            for (0..other.limbs.len) |j| {
                const product = @as(u128, self.limbs[i]) * @as(u128, other.limbs[j]) +
                    @as(u128, result[i + j]) + @as(u128, carry);
                result[i + j] = @truncate(product);
                carry = @truncate(product >> 64);
            }
            result[i + other.limbs.len] +%= carry;
        }

        return normalizeOwned(self.allocator, result);
    }

    fn rem(self: *const BigUint, divisor: *const BigUint) !BigUint {
        if (divisor.isZero()) return error.DivisionByZero;
        if (self.cmp(divisor) == .lt) return self.clone();

        var remainder = try BigUint.zero(self.allocator);
        errdefer remainder.deinit();

        const total_bits = self.bitLen();
        var bit_index = total_bits;
        while (bit_index != 0) {
            bit_index -= 1;
            try remainder.shiftLeft1();
            if (self.bitAt(bit_index)) remainder.limbs[0] |= 1;
            if (remainder.cmp(divisor) != .lt) {
                const next = try remainder.sub(divisor);
                remainder.deinit();
                remainder = next;
            }
        }

        return remainder;
    }

    fn clone(self: *const BigUint) !BigUint {
        const limbs = try self.allocator.dupe(u64, self.limbs);
        return .{ .allocator = self.allocator, .limbs = limbs };
    }

    fn shiftLeft1(self: *BigUint) !void {
        var carry: u64 = 0;
        for (self.limbs) |*limb| {
            const new_carry = limb.* >> 63;
            limb.* = (limb.* << 1) | carry;
            carry = new_carry;
        }
        if (carry == 0) return;

        var expanded = try self.allocator.alloc(u64, self.limbs.len + 1);
        @memcpy(expanded[0..self.limbs.len], self.limbs);
        expanded[self.limbs.len] = carry;
        self.allocator.free(self.limbs);
        self.limbs = expanded;
    }

    fn bitLen(self: *const BigUint) usize {
        if (self.isZero()) return 0;
        const top = self.limbs[self.limbs.len - 1];
        return (self.limbs.len - 1) * 64 + (64 - @clz(top));
    }

    fn bitAt(self: *const BigUint, index: usize) bool {
        const limb_index = index / 64;
        const bit_index = index % 64;
        if (limb_index >= self.limbs.len) return false;
        return ((self.limbs[limb_index] >> @intCast(bit_index)) & 1) == 1;
    }

    fn toLeBytes(self: *const BigUint) ![]u8 {
        var bytes = try self.allocator.alloc(u8, self.limbs.len * 8);
        for (self.limbs, 0..) |limb, i| {
            std.mem.writeInt(u64, bytes[i * 8 ..][0..8], limb, .little);
        }
        var trimmed_len = bytes.len;
        while (trimmed_len > 1 and bytes[trimmed_len - 1] == 0) : (trimmed_len -= 1) {}
        if (trimmed_len == bytes.len) return bytes;

        const trimmed = try self.allocator.dupe(u8, bytes[0..trimmed_len]);
        self.allocator.free(bytes);
        return trimmed;
    }
};

fn normalizeOwned(allocator: std.mem.Allocator, limbs: []u64) !BigUint {
    var len = limbs.len;
    while (len > 1 and limbs[len - 1] == 0) : (len -= 1) {}
    if (len == limbs.len) return .{ .allocator = allocator, .limbs = limbs };

    const trimmed = try allocator.dupe(u64, limbs[0..len]);
    allocator.free(limbs);
    return .{ .allocator = allocator, .limbs = trimmed };
}

fn wotsF(pub_seed: []const u8, chain_idx: usize, step_idx: usize, msg: []const u8) [32]u8 {
    var input: [wots_n + 2 + wots_n]u8 = undefined;
    @memcpy(input[0..wots_n], pub_seed);
    input[wots_n] = @truncate(chain_idx);
    input[wots_n + 1] = @truncate(step_idx);
    @memcpy(input[wots_n + 2 ..], msg);

    var out: [32]u8 = undefined;
    Sha256Hasher.hash(&input, &out, .{});
    return out;
}

fn wotsChain(x: []const u8, start_step: usize, steps: usize, pub_seed: []const u8, chain_idx: usize) [32]u8 {
    var current: [32]u8 = undefined;
    @memcpy(&current, x[0..wots_n]);

    var j = start_step;
    while (j < start_step + steps) : (j += 1) {
        current = wotsF(pub_seed, chain_idx, j, &current);
    }
    return current;
}

fn wotsAllDigits(msg_hash: *const [32]u8) [wots_len]usize {
    var digits: [wots_len]usize = undefined;
    var idx: usize = 0;
    var checksum: usize = 0;

    for (msg_hash) |byte| {
        const high = (byte >> 4) & 0x0f;
        const low = byte & 0x0f;
        digits[idx] = high;
        digits[idx + 1] = low;
        checksum += (wots_w - 1) - high;
        checksum += (wots_w - 1) - low;
        idx += 2;
    }

    var remaining = checksum;
    var i = wots_len;
    while (i > wots_len1) {
        i -= 1;
        digits[i] = remaining % wots_w;
        remaining /= wots_w;
    }

    return digits;
}

fn wotsSecretKeyElement(seed: []const u8, index: usize) [32]u8 {
    var input: [wots_n + 4]u8 = undefined;
    @memcpy(input[0..wots_n], seed);
    std.mem.writeInt(u32, input[wots_n .. wots_n + 4], @intCast(index), .big);

    var out: [32]u8 = undefined;
    Sha256Hasher.hash(&input, &out, .{});
    return out;
}

fn wotsPublicKeyFromSeed(seed: []const u8, pub_seed: []const u8) [64]u8 {
    var endpoints: [wots_len * wots_n]u8 = undefined;
    for (0..wots_len) |i| {
        const sk_element = wotsSecretKeyElement(seed, i);
        const endpoint = wotsChain(&sk_element, 0, wots_w - 1, pub_seed, i);
        @memcpy(endpoints[i * wots_n ..][0..wots_n], &endpoint);
    }

    var root: [32]u8 = undefined;
    Sha256Hasher.hash(&endpoints, &root, .{});

    var pk: [64]u8 = undefined;
    @memcpy(pk[0..32], pub_seed);
    @memcpy(pk[32..64], &root);
    return pk;
}

fn wotsSignDeterministic(message: []const u8, seed: []const u8, pub_seed: []const u8) [wots_len * wots_n]u8 {
    var msg_hash: [32]u8 = undefined;
    Sha256Hasher.hash(message, &msg_hash, .{});
    const digits = wotsAllDigits(&msg_hash);

    var sig: [wots_len * wots_n]u8 = undefined;
    for (0..wots_len) |i| {
        const sk_element = wotsSecretKeyElement(seed, i);
        const element = wotsChain(&sk_element, 0, digits[i], pub_seed, i);
        @memcpy(sig[i * wots_n ..][0..wots_n], &element);
    }
    return sig;
}

fn isIdentityPoint(point: []const u8) bool {
    if (point.len != 64) return false;
    for (point) |byte| {
        if (byte != 0) return false;
    }
    return true;
}

fn parsePoint(point: []const u8) !std.crypto.ecc.Secp256k1 {
    if (point.len != 64) return error.InvalidPointEncoding;
    if (isIdentityPoint(point)) return std.crypto.ecc.Secp256k1.identityElement;

    var sec1 = [_]u8{0} ** 65;
    sec1[0] = 0x04;
    @memcpy(sec1[1..65], point);
    return std.crypto.ecc.Secp256k1.fromSec1(&sec1);
}

fn serializePoint(point: std.crypto.ecc.Secp256k1) base.Point {
    if (point.equivalent(std.crypto.ecc.Secp256k1.identityElement)) {
        return dupeBytes(&([_]u8{0} ** 64));
    }
    const sec1 = point.toUncompressedSec1();
    return dupeBytes(sec1[1..65]);
}

fn scalarBytesFromI64(value: i64) [32]u8 {
    var out = [_]u8{0} ** 32;
    std.mem.writeInt(u64, out[24..32], unsignedAbs(value), .big);
    return out;
}

test "sign fixtures round trip through checkSig" {
    const sig = signTestMessage(test_keys.ALICE);
    defer freeIfOwned(sig);

    try std.testing.expect(checkSig(sig, test_keys.ALICE.pubKey));
    try std.testing.expect(!checkSig(sig, test_keys.BOB.pubKey));
}

test "fixture private keys derive the published compressed pubkeys" {
    const fixtures = [_]test_keys.TestKeyPair{
        test_keys.ALICE,
        test_keys.BOB,
        test_keys.CHARLIE,
    };

    for (fixtures) |fixture| {
        const secret_key = try parseFixtureSecretKey(fixture.privKey);
        const key_pair = try Secp256k1Ecdsa.KeyPair.fromSecretKey(secret_key);
        const derived_pub_key = key_pair.public_key.toCompressedSec1();
        try std.testing.expectEqualSlices(u8, fixture.pubKey, &derived_pub_key);
    }
}

test "signTestMessage matches the known alice fixture signature" {
    const expected = [_]u8{
        0x30, 0x45, 0x02, 0x21, 0x00, 0xe2, 0xaa, 0x12,
        0x65, 0xce, 0x57, 0xf5, 0x4b, 0x98, 0x1f, 0xfc,
        0x6a, 0x5f, 0x3d, 0x22, 0x9e, 0x90, 0x8d, 0x77,
        0x72, 0xfc, 0xeb, 0x75, 0xa5, 0x0c, 0x8c, 0x2d,
        0x60, 0x76, 0x31, 0x3d, 0xf0, 0x02, 0x20, 0x60,
        0x7d, 0xbc, 0xa2, 0xf9, 0xf6, 0x95, 0x43, 0x8b,
        0x49, 0xee, 0xfe, 0xa4, 0xe4, 0x45, 0x66, 0x4c,
        0x74, 0x01, 0x63, 0xaf, 0x8b, 0x62, 0xb1, 0x37,
        0x3f, 0x87, 0xd5, 0x0e, 0xb6, 0x44, 0x17,
    };

    const sig = signTestMessage(test_keys.ALICE);
    defer freeIfOwned(sig);
    try std.testing.expectEqualSlices(u8, &expected, sig);
}

test "checkSig accepts a trailing sighash byte" {
    const base_sig = signTestMessage(test_keys.ALICE);
    defer freeIfOwned(base_sig);

    var with_sighash = std.heap.page_allocator.alloc(u8, base_sig.len + 1) catch @panic("OOM");
    defer std.heap.page_allocator.free(with_sighash);
    @memcpy(with_sighash[0..base_sig.len], base_sig);
    with_sighash[base_sig.len] = 0x41;

    try std.testing.expect(checkSig(with_sighash, test_keys.ALICE.pubKey));
}

test "hash160 matches fixture hashes" {
    const alice_hash = hash160(test_keys.ALICE.pubKey);
    defer freeIfOwned(alice_hash);
    const bob_hash = hash160(test_keys.BOB.pubKey);
    defer freeIfOwned(bob_hash);
    const charlie_hash = hash160(test_keys.CHARLIE.pubKey);
    defer freeIfOwned(charlie_hash);

    try std.testing.expectEqualSlices(u8, test_keys.ALICE.pubKeyHash, alice_hash);
    try std.testing.expectEqualSlices(u8, test_keys.BOB.pubKeyHash, bob_hash);
    try std.testing.expectEqualSlices(u8, test_keys.CHARLIE.pubKeyHash, charlie_hash);
}

test "bytesEq compares byte content explicitly" {
    try std.testing.expect(bytesEq("abc", "abc"));
    try std.testing.expect(!bytesEq("abc", "abd"));
    try std.testing.expect(bytesEq(&.{}, &.{}));
}

test "mock preimage extractors round trip" {
    const expected_hash = hash256("prevouts");
    defer freeIfOwned(expected_hash);
    const output_hash = hash256("outputs");
    defer freeIfOwned(output_hash);

    const preimage = mockPreimage(.{
        .hashPrevouts = expected_hash,
        .outpoint = "outpoint-data",
        .outputHash = output_hash,
        .locktime = 500,
    });
    defer freeIfOwned(preimage);

    const extracted_hash = extractHashPrevouts(preimage);
    defer freeIfOwned(extracted_hash);
    try std.testing.expect(std.mem.eql(u8, extracted_hash, expected_hash));
    try std.testing.expectEqual(@as(i64, 500), extractLocktime(preimage));
}

test "num2bin and bin2num follow signed magnitude semantics" {
    const cases = [_]struct {
        value: i64,
        size: i64,
        expected: []const u8,
    }{
        .{ .value = 0, .size = 0, .expected = &.{} },
        .{ .value = 0, .size = 4, .expected = &[_]u8{ 0, 0, 0, 0 } },
        .{ .value = 1, .size = 1, .expected = &[_]u8{0x01} },
        .{ .value = -1, .size = 1, .expected = &[_]u8{0x81} },
        .{ .value = -1, .size = 4, .expected = &[_]u8{ 0x01, 0x00, 0x00, 0x80 } },
        .{ .value = 128, .size = 2, .expected = &[_]u8{ 0x80, 0x00 } },
        .{ .value = -128, .size = 2, .expected = &[_]u8{ 0x80, 0x80 } },
    };

    for (cases) |case| {
        const encoded = num2bin(case.value, case.size);
        defer freeIfOwned(encoded);
        try std.testing.expectEqualSlices(u8, case.expected, encoded);
        try std.testing.expectEqual(case.value, bin2num(encoded));
    }

    try std.testing.expectEqual(@as(i64, 0), bin2num(&[_]u8{0x80}));
}

test "safemod keeps the dividend sign" {
    try std.testing.expectEqual(@as(i64, -1), safemod(-7, 3));
    try std.testing.expectEqual(@as(i64, 1), safemod(7, 3));
    try std.testing.expectEqual(@as(i64, 0), safemod(7, 0));
}

test "sha256Compress matches known abc hash" {
    var block = [_]u8{0} ** 64;
    @memcpy(block[0..3], "abc");
    std.mem.writeInt(u64, block[56..64], 24, .big);

    const compressed = sha256Compress(sha256_initial_state[0..], &block);
    defer freeIfOwned(compressed);
    const expected = sha256("abc");
    defer freeIfOwned(expected);

    try std.testing.expectEqualSlices(u8, expected, compressed);
}

test "sha256Finalize matches standard sha256 for one and two block messages" {
    const short = sha256Finalize(sha256_initial_state[0..], "abc", 24);
    defer freeIfOwned(short);
    const short_expected = sha256("abc");
    defer freeIfOwned(short_expected);
    try std.testing.expectEqualSlices(u8, short_expected, short);

    const empty = sha256Finalize(sha256_initial_state[0..], "", 0);
    defer freeIfOwned(empty);
    const empty_expected = sha256("");
    defer freeIfOwned(empty_expected);
    try std.testing.expectEqualSlices(u8, empty_expected, empty);

    const long_message = "dd" ** 100;
    const long_hash = sha256Finalize(sha256_initial_state[0..], long_message[0..], 800);
    defer freeIfOwned(long_hash);
    const expected_long_hash = sha256(long_message[0..]);
    defer freeIfOwned(expected_long_hash);
    try std.testing.expectEqualSlices(u8, expected_long_hash, long_hash);
}

test "blake3 helpers follow the single block runtime semantics" {
    const expected_abc = [_]u8{
        0x6f, 0x98, 0x71, 0xb5, 0xd6, 0xe8, 0x0f, 0xc8,
        0x82, 0xe7, 0xbb, 0x57, 0x85, 0x7f, 0x8b, 0x27,
        0x9c, 0xdc, 0x22, 0x96, 0x64, 0xea, 0xb9, 0x38,
        0x2d, 0x28, 0x38, 0xdb, 0xf7, 0xd8, 0xa2, 0x0d,
    };

    const hashed = blake3Hash("abc");
    defer freeIfOwned(hashed);
    try std.testing.expectEqualSlices(u8, &expected_abc, hashed);

    var block = [_]u8{0} ** 64;
    @memcpy(block[0..3], "abc");
    const compressed = blake3Compress(blake3_iv_bytes[0..], &block);
    defer freeIfOwned(compressed);
    try std.testing.expectEqualSlices(u8, hashed, compressed);
}

test "ec helpers use real secp256k1 arithmetic" {
    const g = ecMulGen(1);
    defer freeIfOwned(g);
    try std.testing.expectEqual(@as(usize, 64), g.len);
    try std.testing.expect(ecOnCurve(g));

    const doubled_via_add = ecAdd(g, g);
    defer freeIfOwned(doubled_via_add);
    const doubled_via_mul = ecMul(g, 2);
    defer freeIfOwned(doubled_via_mul);
    const doubled_via_gen = ecMulGen(2);
    defer freeIfOwned(doubled_via_gen);

    try std.testing.expectEqualSlices(u8, doubled_via_add, doubled_via_mul);
    try std.testing.expectEqualSlices(u8, doubled_via_add, doubled_via_gen);

    const neg = ecNegate(g);
    defer freeIfOwned(neg);
    try std.testing.expect(ecOnCurve(neg));

    const identity = ecAdd(g, neg);
    defer freeIfOwned(identity);
    try std.testing.expectEqualSlices(u8, &([_]u8{0} ** 64), identity);
    try std.testing.expect(ecOnCurve(identity));

    const compressed = ecEncodeCompressed(g);
    defer freeIfOwned(compressed);
    try std.testing.expectEqual(@as(usize, 33), compressed.len);
    try std.testing.expect(compressed[0] == 0x02 or compressed[0] == 0x03);
}

test "ec small-value point helpers round trip" {
    const p = ecMakePoint(12345, -67890);
    defer freeIfOwned(p);

    try std.testing.expectEqual(@as(i64, 12345), ecPointX(p));
    try std.testing.expectEqual(@as(i64, -67890), ecPointY(p));
    try std.testing.expect(!ecOnCurve(p));
}

test "verifyWOTS accepts a valid deterministic signature" {
    const seed = [_]u8{0x42} ** 32;
    const pub_seed = [_]u8{0x13} ** 32;
    const pk = wotsPublicKeyFromSeed(&seed, &pub_seed);
    const sig = wotsSignDeterministic("hello WOTS+", &seed, &pub_seed);

    try std.testing.expect(verifyWOTS("hello WOTS+", &sig, &pk));
    try std.testing.expect(!verifyWOTS("wrong message", &sig, &pk));
}

test "verifyRabinSig accepts a trivial valid signature construction" {
    const modulus = [_]u8{0xfb}; // 251, little-endian
    var hash_bytes: [32]u8 = undefined;
    Sha256Hasher.hash("oracle-message", &hash_bytes, .{});

    var hash_bn = try BigUint.fromLeBytes(std.heap.page_allocator, &hash_bytes);
    defer hash_bn.deinit();
    var modulus_bn = try BigUint.fromLeBytes(std.heap.page_allocator, &modulus);
    defer modulus_bn.deinit();
    var padding_bn = try hash_bn.rem(&modulus_bn);
    defer padding_bn.deinit();
    const padding = try padding_bn.toLeBytes();
    defer std.heap.page_allocator.free(padding);

    try std.testing.expect(verifyRabinSig("oracle-message", &[_]u8{0x00}, padding, &modulus));
    try std.testing.expect(!verifyRabinSig("wrong-message", &[_]u8{0x00}, padding, &modulus));
}

test "unsupported SLHDSA variants still fail closed" {
    try std.testing.expect(!verifyRabinSig("msg", "sig", "pad", ""));
    try std.testing.expect(!verifyWOTS("msg", "sig", "pub"));
    try std.testing.expect(!verifySLHDSA_SHA2_128s("msg", "sig", "pub"));
    try std.testing.expect(!verifySLHDSA_SHA2_256f("msg", "sig", "pub"));
}
