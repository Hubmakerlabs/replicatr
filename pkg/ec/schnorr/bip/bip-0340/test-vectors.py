import sys
from reference import *


def is_square(x):
    return int(pow(x, (p - 1) // 2, p)) == 1


def has_square_y(P):
    """Determine if P has a square Y coordinate. Used in an earlier draft of BIP340."""
    assert not is_infinite(P)
    return is_square(P[1])


def vector0():
    seckey = bytes_from_int(3)
    msg = bytes_from_int(0)
    aux_rand = bytes_from_int(0)
    sig = schnorr_sign(msg, seckey, aux_rand)
    pubkey = pubkey_gen(seckey)

    # We should have at least one test vector where the seckey needs to be
    # negated and one where it doesn't. In this one the seckey doesn't need to
    # be negated.
    x = int_from_bytes(seckey)
    P = point_mul(G, x)
    assert (y(P) % 2 == 0)

    # For historical reasons (pubkey tiebreaker was squareness and not evenness)
    # we should have at least one test vector where the the point reconstructed
    # from the public key has a square and one where it has a non-square Y
    # coordinate. In this one Y is non-square.
    pubkey_point = lift_x(pubkey)
    assert (not has_square_y(pubkey_point))

    # For historical reasons (R tiebreaker was squareness and not evenness)
    # we should have at least one test vector where the the point reconstructed
    # from the R.x coordinate has a square and one where it has a non-square Y
    # coordinate. In this one Y is non-square.
    R = lift_x(sig[0:32])
    assert (not has_square_y(R))

    return seckey, pubkey, aux_rand, msg, sig, "TRUE", None


def vector1():
    seckey = bytes_from_int(0xB7E151628AED2A6ABF7158809CF4F3C762E7160F38B4DA56A784D9045190CFEF)
    msg = bytes_from_int(0x243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89)
    aux_rand = bytes_from_int(1)

    sig = schnorr_sign(msg, seckey, aux_rand)

    # The point reconstructed from the R.x coordinate has a square Y coordinate.
    R = lift_x(sig[0:32])
    assert (has_square_y(R))

    return seckey, pubkey_gen(seckey), aux_rand, msg, sig, "TRUE", None


def vector2():
    seckey = bytes_from_int(0xC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B14E5C9)
    msg = bytes_from_int(0x7E2D58D8B3BCDF1ABADEC7829054F90DDA9805AAB56C77333024B9D0A508B75C)
    aux_rand = bytes_from_int(0xC87AA53824B4D7AE2EB035A2B5BBBCCC080E76CDC6D1692C4B0B62D798E6D906)
    sig = schnorr_sign(msg, seckey, aux_rand)

    # The point reconstructed from the public key has a square Y coordinate.
    pubkey = pubkey_gen(seckey)
    pubkey_point = lift_x(pubkey)
    assert (has_square_y(pubkey_point))

    # This signature vector would not verify if the implementer checked the
    # evenness of the X coordinate of R instead of the Y coordinate.
    R = lift_x(sig[0:32])
    assert (R[0] % 2 == 1)

    return seckey, pubkey, aux_rand, msg, sig, "TRUE", None


def vector3():
    seckey = bytes_from_int(0x0B432B2677937381AEF05BB02A66ECD012773062CF3FA2549E44F58ED2401710)

    # Need to negate this seckey before signing
    x = int_from_bytes(seckey)
    P = point_mul(G, x)
    assert (y(P) % 2 != 0)

    msg = bytes_from_int(0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF)
    aux_rand = bytes_from_int(0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF)

    sig = schnorr_sign(msg, seckey, aux_rand)
    return seckey, pubkey_gen(seckey), aux_rand, msg, sig, "TRUE", "test fails if msg is reduced modulo p or n"


# Signs with a given nonce. This can be INSECURE and is only INTENDED FOR
# GENERATING TEST VECTORS. Results in an invalid signature if y(kG) is not
# even.
def insecure_schnorr_sign_fixed_nonce(msg, seckey0, k):
    if len(msg) != 32:
        raise ValueError('The message must be a 32-byte array.')
    seckey0 = int_from_bytes(seckey0)
    if not (1 <= seckey0 <= n - 1):
        raise ValueError('The secret key must be an integer in the range 1..n-1.')
    P = point_mul(G, seckey0)
    seckey = seckey0 if has_even_y(P) else n - seckey0
    R = point_mul(G, k)
    e = int_from_bytes(tagged_hash("BIP0340/challenge", bytes_from_point(R) + bytes_from_point(P) + msg)) % n
    return bytes_from_point(R) + bytes_from_int((k + e * seckey) % n)


# Creates a singature with a small x(R) by using k = -1/2
def vector4():
    one_half = n - 0x7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0
    seckey = bytes_from_int(0x763758E5CBEEDEE4F7D3FC86F531C36578933228998226672F13C4F0EBE855EB)
    msg = bytes_from_int(0x4DF3C3F68FCC83B27E9D42C90431A72499F17875C81A599B566C9889B9696703)
    sig = insecure_schnorr_sign_fixed_nonce(msg, seckey, one_half)
    return None, pubkey_gen(seckey), None, msg, sig, "TRUE", None


default_seckey = bytes_from_int(0xB7E151628AED2A6ABF7158809CF4F3C762E7160F38B4DA56A784D9045190CFEF)
default_msg = bytes_from_int(0x243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89)
default_aux_rand = bytes_from_int(0xC87AA53824B4D7AE2EB035A2B5BBBCCC080E76CDC6D1692C4B0B62D798E6D906)


# Public key is not on the curve
def vector5():
    # This creates a dummy signature that doesn't have anything to do with the
    # public key.
    seckey = default_seckey
    msg = default_msg
    sig = schnorr_sign(msg, seckey, default_aux_rand)

    pubkey = bytes_from_int(0xEEFDEA4CDB677750A420FEE807EACF21EB9898AE79B9768766E4FAA04A2D4A34)
    assert (lift_x(pubkey) is None)

    return None, pubkey, None, msg, sig, "FALSE", "public key not on the curve"


def vector6():
    seckey = default_seckey
    msg = default_msg
    k = 6
    sig = insecure_schnorr_sign_fixed_nonce(msg, seckey, k)

    # Y coordinate of R is not even
    R = point_mul(G, k)
    assert (not has_even_y(R))

    return None, pubkey_gen(seckey), None, msg, sig, "FALSE", "has_even_y(R) is false"


def vector7():
    seckey = default_seckey
    msg = int_from_bytes(default_msg)
    neg_msg = bytes_from_int(n - msg)
    sig = schnorr_sign(neg_msg, seckey, default_aux_rand)
    return None, pubkey_gen(seckey), None, bytes_from_int(msg), sig, "FALSE", "negated message"


def vector8():
    seckey = default_seckey
    msg = default_msg
    sig = schnorr_sign(msg, seckey, default_aux_rand)
    sig = sig[0:32] + bytes_from_int(n - int_from_bytes(sig[32:64]))
    return None, pubkey_gen(seckey), None, msg, sig, "FALSE", "negated s value"


def bytes_from_point_inf0(P):
    if P is None:
        return bytes_from_int(0)
    return bytes_from_int(P[0])


def vector9():
    seckey = default_seckey
    msg = default_msg

    # Override bytes_from_point in schnorr_sign to allow creating a signature
    # with k = 0.
    k = 0
    bytes_from_point_tmp = bytes_from_point.__code__
    bytes_from_point.__code__ = bytes_from_point_inf0.__code__
    sig = insecure_schnorr_sign_fixed_nonce(msg, seckey, k)
    bytes_from_point.__code__ = bytes_from_point_tmp

    return (None, pubkey_gen(seckey), None, msg, sig, "FALSE",
            "sG - eP is infinite. Test fails in single verification if has_even_y(inf) is defined as true and x(inf) as 0")


def bytes_from_point_inf1(P):
    if P == None:
        return bytes_from_int(1)
    return bytes_from_int(P[0])


def vector10():
    seckey = default_seckey
    msg = default_msg

    # Override bytes_from_point in schnorr_sign to allow creating a signature
    # with k = 0.
    k = 0
    bytes_from_point_tmp = bytes_from_point.__code__
    bytes_from_point.__code__ = bytes_from_point_inf1.__code__
    sig = insecure_schnorr_sign_fixed_nonce(msg, seckey, k)
    bytes_from_point.__code__ = bytes_from_point_tmp

    return (None, pubkey_gen(seckey), None, msg, sig, "FALSE",
            "sG - eP is infinite. Test fails in single verification if has_even_y(inf) is defined as true and x(inf) as 1")


# It's cryptographically impossible to create a test vector that fails if run
# in an implementation which merely misses the check that sig[0:32] is an X
# coordinate on the curve. This test vector just increases test coverage.
def vector11():
    seckey = default_seckey
    msg = default_msg
    sig = schnorr_sign(msg, seckey, default_aux_rand)

    # Replace R's X coordinate with an X coordinate that's not on the curve
    x_not_on_curve = bytes_from_int(0x4A298DACAE57395A15D0795DDBFD1DCB564DA82B0F269BC70A74F8220429BA1D)
    assert (lift_x(x_not_on_curve) is None)
    sig = x_not_on_curve + sig[32:64]

    return None, pubkey_gen(seckey), None, msg, sig, "FALSE", "sig[0:32] is not an X coordinate on the curve"


# It's cryptographically impossible to create a test vector that fails if run
# in an implementation which merely misses the check that sig[0:32] is smaller
# than the field size. This test vector just increases test coverage.
def vector12():
    seckey = default_seckey
    msg = default_msg
    sig = schnorr_sign(msg, seckey, default_aux_rand)

    # Replace R's X coordinate with an X coordinate that's equal to field size
    sig = bytes_from_int(p) + sig[32:64]

    return None, pubkey_gen(seckey), None, msg, sig, "FALSE", "sig[0:32] is equal to field size"


# It's cryptographically impossible to create a test vector that fails if run
# in an implementation which merely misses the check that sig[32:64] is smaller
# than the curve order. This test vector just increases test coverage.
def vector13():
    seckey = default_seckey
    msg = default_msg
    sig = schnorr_sign(msg, seckey, default_aux_rand)

    # Replace s with a number that's equal to the curve order
    sig = sig[0:32] + bytes_from_int(n)

    return None, pubkey_gen(seckey), None, msg, sig, "FALSE", "sig[32:64] is equal to curve order"


# Test out of range pubkey
# It's cryptographically impossible to create a test vector that fails if run
# in an implementation which accepts out of range pubkeys because we can't find
# a secret key for such a public key and therefore can not create a signature.
# This test vector just increases test coverage.
def vector14():
    # This creates a dummy signature that doesn't have anything to do with the
    # public key.
    seckey = default_seckey
    msg = default_msg
    sig = schnorr_sign(msg, seckey, default_aux_rand)
    pubkey_int = p + 1
    pubkey = bytes_from_int(pubkey_int)
    assert (lift_x(pubkey) is None)
    # If an implementation would reduce a given public key modulo p then the
    # pubkey would be valid
    assert (lift_x(bytes_from_int(pubkey_int % p)) is not None)

    return (
    None, pubkey, None, msg, sig, "FALSE", "public key is not a valid X coordinate because it exceeds the field size")


def varlen_vector(msg_int):
    seckey = bytes_from_int(int(16 * "0340", 16))
    pubkey = pubkey_gen(seckey)
    aux_rand = bytes_from_int(0)
    msg = msg_int.to_bytes((msg_int.bit_length() + 7) // 8, "big")
    sig = schnorr_sign(msg, seckey, aux_rand)
    comment = "message of size %d (added 2022-12)"
    return seckey, pubkey, aux_rand, msg, sig, "TRUE", comment % len(msg)


vector15 = lambda: varlen_vector(0)
vector16 = lambda: varlen_vector(0x11)
vector17 = lambda: varlen_vector(0x0102030405060708090A0B0C0D0E0F1011)
vector18 = lambda: varlen_vector(int(100 * "99", 16))

vectors = [
    vector0(),
    vector1(),
    vector2(),
    vector3(),
    vector4(),
    vector5(),
    vector6(),
    vector7(),
    vector8(),
    vector9(),
    vector10(),
    vector11(),
    vector12(),
    vector13(),
    vector14(),
    vector15(),
    vector16(),
    vector17(),
    vector18(),
]


# Converts the byte strings of a test vector into hex strings
def bytes_to_hex(seckey, pubkey, aux_rand, msg, sig, result, comment):
    return (seckey.hex().upper() if seckey is not None else None, pubkey.hex().upper(),
            aux_rand.hex().upper() if aux_rand is not None else None, msg.hex().upper(), sig.hex().upper(), result,
            comment)


vectors = list(
    map(lambda vector: bytes_to_hex(vector[0], vector[1], vector[2], vector[3], vector[4], vector[5], vector[6]),
        vectors))


def print_csv(vectors):
    writer = csv.writer(sys.stdout)
    writer.writerow(
        ("index", "secret key", "public key", "aux_rand", "message", "signature", "verification result", "comment"))
    for (i, v) in enumerate(vectors):
        writer.writerow((i,) + v)


print_csv(vectors)
