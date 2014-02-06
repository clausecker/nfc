// API to interface with the libnfc

package nfc

// maximum length for an NFC connection string
const BUFSIZE_CONNSTRING = 1024

// properties
const (
	TIMEOUT_COMMAND = iota
	TIMEOUT_ATR
	TIMEOUT_COM
	TIMEOUT_CRC
	HANDLE_CRC
	HANDLE_PARITY
	ACTIVATE_FIELD
	ACTIVATE_CRYPTO1
	INFINITE_SELECT
	ACCEPT_INVALID_FRAMES
	ACCEPT_MULTIPLE_FRAMES
	AUTO_ISO14443_4
	EASY_FRAMING
	FORCE_ISO14443_A
	FORCE_ISO14443_B
	FORCE_SPEED_106
)

// NFC D.E.P. (Data Exchange Protocol) active/passive mode
const (
	NDM_UNDEFINED = iota
	NDM_PASSIVE
	NDM_ACTIVE
)

// an error as reported by various methods of Device. If device returns an error
// that is not castable to Error, something outside on the Go side went wrong.
type Error int

// replicates the behavior of nfc_errstr.
func (e Error) Error() string {
	if errorMessages[int(e)] == "" {
		return "Unknown error"
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
	ERFTRANS     = -20 // Error while RF transmission
	EMFCAUTHFAIL = -30 // MIFARE Classic: authentication failed
	ESOFT        = -80 // Software error (allocation, file/pipe creation, etc.)
	ECHIP        = -90 // Device's internal chip error
)

// replicate error messages here because the libnfc is incapable of giving
// direct access to the error strings. Stupidly, only the error string for the
// error code of an nfc_device can be read out.
var errorMessages = map[int]string{
	SUCCESS:      "Success",
	EIO:          "Input / Output Error",
	EINVARG:      "Invalid argument(s)",
	EDEVNOTSUPP:  "Not Supported by Device",
	ENOTSUCHDEV:  "No Such Device",
	EOVFLOW:      "Buffer Overflow",
	ETIMEOUT:     "Timeout",
	EOPABORTED:   "Operation Aborted",
	ENOTIMPL:     "Not (yet) Implemented",
	ETGRELEASED:  "Target Released",
	EMFCAUTHFAIL: "Mifare Authentication Failed",
	ERFTRANS:     "RF Transmission Error",
	ECHIP:        "Device's Internal Chip Error",
}

// the global library context
var theContext *context

// open a connection to an NFC device. If conn is "", the first available device
// will be used. If this operation fails, check the log on stderr for more
// details as the libnfc is not particulary verbose to us.
//
// Depending on the desired operation mode, the device needs to be configured
// by using InitiatorInit() or TargetInit(), optionally followed by manual
// tuning of the parameters if the default parameters are not suiting your
// goals.
func Open(conn string) (*Device, error) {
	return theContext.open(conn)
}
