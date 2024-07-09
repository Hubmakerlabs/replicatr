package hex

import "encoding/hex"

type InvalidByteError = hex.InvalidByteError

var Enc = hex.EncodeToString
var Dec = hex.DecodeString
var DecLen = hex.DecodedLen
var Decode = hex.Decode
