// Copyright (c) 2014, 2020, 2024 Robert Clausecker <fuzxxl@gmail.com>
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU Lesser General Public License as published by the
// Free Software Foundation, version 3.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>

// Package nfc wraps the libnfc to provide an API for Go. Most documentation
// was taken unchanged from the documentation inside the libnfc. Some functions
// and names have been altered to fit the conventions and idioms used in Go.
//
// To use this package, obtain and install libnfc.  By default, pkg-config is
// used to find and link libnfc.  If you cannot use pkg-config, you can compile
// with build-tag nopkgconfig or no_pkgconfig to instead link with -lnfc.
package nfc

import "fmt"
import "strconv"

// Maximum length for an NFC connection string
const BufsizeConnstring = 1024

// Properties for (*Device).SetPropertyInt() and (*Device).SetPropertyBool().
const (
	// Default command processing timeout
	// Property value's (duration) unit is ms and 0 means no timeout (infinite).
	// Default value is set by driver layer
	TimeoutCommand = iota

	// Timeout between ATR_REQ and ATR_RES
	// When the device is in initiator mode, a target is considered as mute
	// if no valid ATR_RES is received within this timeout value.
	// Default value for this property is 103 ms on PN53x based devices.
	TimeoutATR

	// Timeout value to give up reception from the target in case of no answer.
	// Default value for this property is 52 ms).
	TimeoutCom

	// Let the PN53X chip handle the CRC bytes. This means that the chip
	// appends the CRC bytes to the frames that are transmitted. It will
	// parse the last bytes from received frames as incoming CRC bytes. They
	// will be verified against the used modulation and protocol. If a frame
	// is expected with incorrect CRC bytes this option should be disabled.
	// Example frames where this is useful are the ATQA and UID+BCC that are
	// transmitted without CRC bytes during the anti-collision phase of the
	// ISO14443-A protocol.
	HandleCRC

	// Parity bits in the network layer of ISO14443-A are by default
	// generated and validated in the PN53X chip. This is a very convenient
	// feature. On certain times though it is useful to get full control of
	// the transmitted data. The proprietary MIFARE Classic protocol uses
	// for example custom (encrypted) parity bits. For interoperability it
	// is required to be completely compatible, including the arbitrary
	// parity bits. When this option is disabled, the functions to
	// communicating bits should be used.
	HandleParity

	// This option can be used to enable or disable the electronic field of
	// the NFC device.
	ActivateField

	// The internal CRYPTO1 co-processor can be used to transmit messages
	// encrypted. This option is automatically activated after a successful
	// MIFARE Classic authentication.
	ActivateCrypto1

	// The default configuration defines that the PN53X chip will try
	// indefinitely to invite a tag in the field to respond. This could be
	// desired when it is certain a tag will enter the field. On the other
	// hand, when this is uncertain, it will block the application. This
	// option could best be compared to the (NON)BLOCKING option used by
	// (socket)network programming.
	InfiniteSelect

	// If this option is enabled, frames that carry less than 4 bits are
	// allowed. According to the standards these frames should normally be
	// handles as invalid frames.
	AcceptInvalidFrames

	// If the NFC device should only listen to frames, it could be useful to
	// let it gather multiple frames in a sequence. They will be stored in
	// the internal FIFO of the PN53X chip. This could be retrieved by using
	// the receive data functions. Note that if the chip runs out of bytes
	// (FIFO = 64 bytes long), it will overwrite the first received frames,
	// so quick retrieving of the received data is desirable.
	AcceptMultipleFrames

	// This option can be used to enable or disable the auto-switching mode
	// to ISO14443-4 is device is compliant.
	// In initiator mode, it means that NFC chip will send RATS
	// automatically when select and it will automatically poll for
	// ISO14443-4 card when ISO14443A is requested.
	// In target mode, with a NFC chip compliant (ie. PN532), the chip will
	// emulate a 14443-4 PICC using hardware capability.
	AutoISO14443_4

	// Use automatic frames encapsulation and chaining.
	EasyFraming

	// Force the chip to switch in ISO14443-A
	ForceISO14443a

	// Force the chip to switch in ISO14443-B
	ForceISO14443b

	// Force the chip to run at 106 kbps
	ForceSpeed106
)

// NFC modulation types
const (
	ISO14443a = iota + 1
	Jewel
	ISO14443b
	ISO14443bi   // pre-ISO14443B aka ISO/IEC 14443 B' or Type B'
	ISO14443b2sr // ISO14443-2B ST SRx
	ISO14443b2ct // ISO14443-2B ASK CTx
	Felica
	DEP
	Barcode         // Thinfilm NFC Barcode
	ISO14443biClass // HID iClass 14443B mode
)

// NFC baud rates. UNDEFINED is also a valid baud rate, albeit defined
// further below.
const (
	Nbr106 = iota + 1
	Nbr212
	Nbr424
	Nbr847
)

// NFC modes. An NFC device can either be a target or an initiator.
const (
	TargetMode = iota
	InitiatorMode
)

// Names corresponding to the NFC modulation types
var ModulationTypes = [...]string{
	0:               "0",
	ISO14443a:       "ISO14443a",
	Jewel:           "Jewel",
	ISO14443b:       "ISO14443b",
	ISO14443bi:      "ISO14443bi",
	ISO14443b2sr:    "ISO14443b2sr",
	ISO14443b2ct:    "ISO14443b2ct",
	Felica:          "Felica",
	DEP:             "DEP",
	Barcode:         "Barcode",
	ISO14443biClass: "ISO14443biClass",
}

// Baud rates corresponding to the baud rate enumeration constants
var BaudRates = [...]string{
	Undefined: "Undefined",
	Nbr106:    "106",
	Nbr212:    "212",
	Nbr424:    "424",
	Nbr847:    "847",
}

// NFC modulation structure. Use the supplied constants.
type Modulation struct {
	Type     int
	BaudRate int
}

// Print a modulation in a human readable manner.
func (m Modulation) String() string {
	var typeStr, brStr string

	if 0 <= m.Type && m.Type < len(ModulationTypes) {
		typeStr = ModulationTypes[m.Type]
	} else {
		typeStr = strconv.Itoa(m.Type)
	}

	if 0 <= m.BaudRate && m.BaudRate < len(BaudRates) {
		brStr = BaudRates[m.BaudRate] + " kbps"
	} else {
		brStr = strconv.Itoa(m.BaudRate)
	}

	return typeStr + " (" + brStr + ")"
}

// Print a modulation in Go syntax.
func (m Modulation) GoString() string {
	var typeStr, brStr string

	if 0 <= m.Type && m.Type < len(ModulationTypes) {
		typeStr = "nfc." + ModulationTypes[m.Type]
	} else {
		typeStr = strconv.Itoa(m.Type)
	}

	if m.BaudRate == 0 {
		brStr = "nfc.Undefined"
	} else if 0 < m.BaudRate && m.BaudRate < len(BaudRates) {
		brStr = "nfc.Nbr" + BaudRates[m.BaudRate]
	} else {
		brStr = strconv.Itoa(m.BaudRate)
	}

	return "nfc.Modulation{Type: " + typeStr + ", BaudRate: " + brStr + "}"
}

// An error as reported by various methods of Device. If device returns an error
// that is not castable to Error, something outside on the Go side went wrong.
type Error int

// Returns the same strings as nfc_errstr except if the error is not among the
// known errors. Instead of reporting an "Unknown error", Error() will return
// something like "Error -123".
func (e Error) Error() string {
	if errorMessages[int(e)] == "" {
		return fmt.Sprintf("Error %d", int(e))
	}

	return errorMessages[int(e)]
}

// Error codes. Casted to Error, these yield all possible errors this package
// provides. Use nfc.Error(errorcode).Error() to get a descriptive string for an
// error code.
const (
	SUCCESS      = 0   // Success (no error)
	EIO          = -1  // Input / output error, device may not be usable anymore without re-open it
	EINVARG      = -2  // Invalid argument(s)
	EDEVNOTSUPP  = -3  // Operation not supported by device
	ENOTSUCHDEV  = -4  // No such device
	EOVFLOW      = -5  // Buffer overflow
	ETIMEOUT     = -6  // Operation timed out
	EOPABORTED   = -7  // Operation aborted (by user)
	ENOTIMPL     = -8  // Not (yet) implemented
	ETGRELEASED  = -10 // Target released
	ERFTRANS     = -20 // Error during RF transmission
	EMFCAUTHFAIL = -30 // MIFARE Classic: authentication failed
	ESOFT        = -80 // Software error (allocation, file/pipe creation, etc.)
	ECHIP        = -90 // Device's internal chip error
)

// replicate error messages here because the libnfc is incapable of giving
// direct access to the error strings. Stupidly, only the error string for the
// error code of an nfc_device can be read out.
var errorMessages = map[int]string{
	SUCCESS:      "success",
	EIO:          "input / output error",
	EINVARG:      "invalid argument(s)",
	EDEVNOTSUPP:  "not supported by device",
	ENOTSUCHDEV:  "no such device",
	EOVFLOW:      "buffer overflow",
	ETIMEOUT:     "timeout",
	EOPABORTED:   "operation aborted",
	ENOTIMPL:     "not (yet) implemented",
	ETGRELEASED:  "target released",
	EMFCAUTHFAIL: "Mifare Classic authentication failed",
	ERFTRANS:     "RF transmission error",
	ESOFT:        "software error",
	ECHIP:        "device's internal chip error",
}

// the global library context
var theContext *context = &context{}
