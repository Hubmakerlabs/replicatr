package taproot

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"mleku.online/git/bech32"
	"mleku.online/git/ec/chaincfg"
)

// AddressSegWit is the base address type for all SegWit addresses.
type AddressSegWit struct {
	hrp            string
	witnessVersion byte
	witnessProgram []byte
}

// AddressTaproot is an Address for a pay-to-taproot (P2TR) output. See BIP 341
// for further details.
type AddressTaproot struct {
	AddressSegWit
}

// NewAddressTaproot returns a new AddressTaproot.
func NewAddressTaproot(witnessProg []byte,
	net *chaincfg.Params) (*AddressTaproot, error) {

	return newAddressTaproot(net.Bech32HRPSegwit, witnessProg)
}

// newAddressWitnessScriptHash is an internal helper function to create an
// AddressWitnessScriptHash with a known human-readable part, rather than
// looking it up through its parameters.
func newAddressTaproot(hrp string,
	witnessProg []byte) (*AddressTaproot, error) {

	// Check for valid program length for witness version 1, which is 32
	// for P2TR.
	if len(witnessProg) != 32 {
		return nil, errors.New("witness program must be 32 bytes for " +
			"p2tr")
	}

	addr := &AddressTaproot{
		AddressSegWit{
			hrp:            strings.ToLower(hrp),
			witnessVersion: 0x01,
			witnessProgram: witnessProg,
		},
	}

	return addr, nil
}

// decodeSegWitAddress parses a bech32 encoded segwit address string and
// returns the witness version and witness program byte representation.
func decodeSegWitAddress(address string) (byte, []byte, error) {
	// Decode the bech32 encoded address.
	_, data, bech32version, e := bech32.DecodeGeneric(address)
	if e != nil {
		return 0, nil, err
	}

	// The first byte of the decoded address is the witness version, it must
	// exist.
	if len(data) < 1 {
		return 0, nil, fmt.Errorf("no witness version")
	}

	// ...and be <= 16.
	version := data[0]
	if version > 16 {
		return 0, nil, fmt.Errorf("invalid witness version: %v", version)
	}

	// The remaining characters of the address returned are grouped into
	// words of 5 bits. In order to restore the original witness program
	// bytes, we'll need to regroup into 8 bit words.
	regrouped, e := bech32.ConvertBits(data[1:], 5, 8, false)
	if e != nil {
		return 0, nil, err
	}

	// The regrouped data must be between 2 and 40 bytes.
	if len(regrouped) < 2 || len(regrouped) > 40 {
		return 0, nil, fmt.Errorf("invalid data length")
	}

	// For witness version 0, address MUST be exactly 20 or 32 bytes.
	if version == 0 && len(regrouped) != 20 && len(regrouped) != 32 {
		return 0, nil, fmt.Errorf("invalid data length for witness "+
			"version 0: %v", len(regrouped))
	}

	// For witness version 0, the bech32 encoding must be used.
	if version == 0 && bech32version != bech32.Version0 {
		return 0, nil, fmt.Errorf("invalid checksum expected bech32 " +
			"encoding for address with witness version 0")
	}

	// For witness version 1, the bech32m encoding must be used.
	if version == 1 && bech32version != bech32.VersionM {
		return 0, nil, fmt.Errorf("invalid checksum expected bech32m " +
			"encoding for address with witness version 1")
	}

	return version, regrouped, nil
}

// encodeSegWitAddress creates a bech32 (or bech32m for SegWit v1) encoded
// address string representation from witness version and witness program.
func encodeSegWitAddress(hrp string, witnessVersion byte,
	witnessProgram []byte) (string, error) {
	// Group the address bytes into 5 bit groups, as this is what is used to
	// encode each character in the address string.
	converted, e := bech32.ConvertBits(witnessProgram, 8, 5, true)
	if e != nil {
		return "", err
	}

	// Concatenate the witness version and program, and encode the resulting
	// bytes using bech32 encoding.
	combined := make([]byte, len(converted)+1)
	combined[0] = witnessVersion
	copy(combined[1:], converted)

	var bech string
	switch witnessVersion {
	case 0:
		bech, e = bech32.Encode(hrp, combined)

	case 1:
		bech, e = bech32.EncodeM(hrp, combined)

	default:
		return "", fmt.Errorf("unsupported witness version %d",
			witnessVersion)
	}
	if e != nil {
		return "", err
	}

	// Check validity by decoding the created address.
	version, program, e := decodeSegWitAddress(bech)
	if e != nil {
		return "", fmt.Errorf("invalid segwit address: %v", e)
	}

	if version != witnessVersion || !bytes.Equal(program, witnessProgram) {
		return "", fmt.Errorf("invalid segwit address")
	}

	return bech, nil
}

// EncodeAddress returns the bech32 (or bech32m for SegWit v1) string encoding
// of an AddressSegWit.
//
// NOTE: This method is part of the Address interface.
func (a *AddressSegWit) EncodeAddress() string {
	str, e := encodeSegWitAddress(
		a.hrp, a.witnessVersion, a.witnessProgram[:],
	)
	if e != nil {
		return ""
	}
	return str
}
