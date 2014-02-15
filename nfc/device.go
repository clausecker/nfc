package nfc

/*
#include <stdlib.h>
#include <nfc/nfc.h>

struct target_listing {
	int count; // is an error code if negative
	nfc_target *entries;
};

// this function works analogeous to list_devices_wrapper but for the function
// nfc_initiator_list_passive_targets.
struct target_listing list_targets_wrapper(nfc_device *device, const nfc_modulation nm) {
	size_t targets_len = 16, actual_count; // 16
	nfc_target *targets = NULL, *targets_tmp;
	struct target_listing  tar;

	// call nfc_list_devices as long as our array might be too short
	for (;;) {
		targets_tmp = realloc(targets, targets_len * sizeof *targets);
		if (targets_tmp == NULL) {
			actual_count = NFC_ESOFT;
			break;
		}

		targets = targets_tmp;
		actual_count = nfc_initiator_list_passive_targets(device, nm, targets, targets_len);

		// also covers the case where actual_count is an error
		if (actual_count < targets_len) break;

		targets_len += 16;
	}

	tar.count = actual_count;
	tar.entries = targets;

	return tar;
}

// Accessing arrays from Go is difficult. We use this helper instead.
nfc_target *index_targets(nfc_target *t, int index) {
	return t + index;
}
*/
import "C"
import "unsafe"
import "errors"
import "fmt"

// NFC device. Copying these structs may cause unintended side effects.
type Device struct {
	d *C.nfc_device
}

// Return a pointer to the wrapped nfc_device. This is useful if you try to use
// this wrapper to wrap other C code that builds onto the libnfc.
func (d *Device) Pointer() uintptr {
	return uintptr(unsafe.Pointer(d.d))
}

// Open an NFC device. See documentation of Open() for more details
func (c *context) open(conn string) (d *Device, err error) {
	c.m.Lock()
	defer c.m.Unlock()
	c.initContext()

	cs, err := newConnstring(conn)
	if err != nil {
		return
	}

	defer cs.Free()

	dev := C.nfc_open(c.c, cs.ptr)

	if dev == nil {
		err = errors.New("Cannot open NFC device")
		return
	}

	d = &Device{dev}
	err = d.lastError()
	return
}

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

// the error returned by the last operation on d. Every function that wraps some
// functions operating on an nfc_device should call this function and return the
// result. This wraps nfc_device_get_last_error.
func (d *Device) lastError() error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	err := Error(C.nfc_device_get_last_error(d.d))

	if err == 0 {
		return nil
	}

	return err
}

// Send data to target then retrieve data from target. n contains received bytes
// count on success, or is meaningless on error. The current implementation will
// return the libnfc error code in case of error, but this is subject to change.
// This function will return EOVFLOW if more bytes are being received than the
// length of rx.
//
// The NFC device (configured as initiator) will transmit the supplied bytes
// (tx) to the target. It waits for the response and stores the received bytes
// in rx. If the received bytes exceed rx, the error status will be NFC_EOVFLOW
// and rx will contain len(rx) received bytes.
//
// If EASY_FRAMING option is disabled the frames will sent and received in raw
// mode: PN53x will not handle input neither output data.
//
// The parity bits are handled by the PN53x chip. The CRC can be generated
// automatically or handled manually. Using this function, frames can be
// communicated very fast via the NFC initiator to the tag.
//
// Tests show that on average this way of communicating is much faster than
// using the regular driver/middle-ware (often supplied by manufacturers).
//
// Warning: The configuration option HANDLE_PARITY must be set to true (the
// default value).
//
// If timeout equals to 0, the function blocks indefinitely (until an error is
// raised or function is completed). If timeout equals to -1, the default
// timeout will be used.
func (d *Device) InitiatorTransceiveBytes(tx, rx []byte, timeout int) (n int, err error) {
	if d.d == nil {
		return ESOFT, errors.New("Device closed")
	}

	txptr := (*C.uint8_t)(&tx[0])
	rxptr := (*C.uint8_t)(&rx[0])

	n = int(C.nfc_initiator_transceive_bytes(
		d.d,
		txptr, C.size_t(len(tx)),
		rxptr, C.size_t(len(rx)),
		C.int(timeout),
	))

	if n < 0 {
		err = Error(n)
	}

	return
}

// Transceive raw bit-frame to a target. n contains the received byte count on
// success, or is meaningless on error. The current implementation will return
// the libnfc error code in case of error, but this is subject to change. If
// txLength is longer than the supplied slice, an error will occur. txPar has to
// have the same length as tx, dito for rxPar and rx. An error will occur if any
// of these invariants do not hold.
//
// tx contains a byte slice of the frame that needs to be transmitted. txLength
// contains its length in bits.
//
// For example the REQA (0x26) command (the first anti-collision command of
// ISO14443-A) must be precise 7 bits long. This is not possible using
// (*Device).InitiatorTransceiveBytes(). With that function you can only
// communicate frames that consist of full bytes. When you send a full byte (8
// bits + 1 parity) with the value of REQA (0x26), a tag will simply not
// respond.
//
// txPar contains a byte slice of the corresponding parity bits needed to send
// per byte.
//
// For example if you send the SELECT_ALL (0x93, 0x20) = [ 10010011, 00100000 ]
// command, you have to supply the following parity bytes (0x01, 0x00) to define
// the correct odd parity bits. This is only an example to explain how it works,
// if you just are sending two bytes with ISO14443-A compliant parity bits you
// better can use the (*Device).InitiatorTransceiveBytes() method.
//
// rx will contain the response from the target. This function will return
// EOVFLOW if more bytes are received than the length of rx. rxPar contains a
// byte slice of the corresponding parity bits.
//
// The NFC device (configured as initiator) will transmit low-level messages
// where only the modulation is handled by the PN53x chip. Construction of the
// frame (data, CRC and parity) is completely done by libnfc.  This can be very
// useful for testing purposes. Some protocols (e.g. MIFARE Classic) require to
// violate the ISO14443-A standard by sending incorrect parity and CRC bytes.
// Using this feature you are able to simulate these frames.
func (d *Device) InitiatorTransceiveBits(tx, txPar []byte, txLength int, rx, rxPar []byte) (n int, err error) {
	if d.d == nil {
		return ESOFT, errors.New("Device closed")
	}

	if len(tx) != len(txPar) || len(rx) != len(rxPar) {
		return ESOFT, errors.New("Invariant doesn't hold")
	}

	if len(tx) < 8*txLength {
		return ESOFT, errors.New("Slice shorter than specified bit count")
	}

	txptr := (*C.uint8_t)(&tx[0])
	txparptr := (*C.uint8_t)(&txPar[0])
	rxptr := (*C.uint8_t)(&rx[0])
	rxparptr := (*C.uint8_t)(&rxPar[0])

	n = int(C.nfc_initiator_transceive_bits(
		d.d,
		txptr, C.size_t(txLength), txparptr,
		rxptr, C.size_t(len(rx)), rxparptr,
	))

	if n < 0 {
		err = Error(n)
	}

	return
}

// Send data to target then retrieve data from target with timing control. n
// contains the received byte count on success, or is meaningless on error. c
// will contain the actual number of cycles waited. The current implementation
// will return the libnfc error code in case of error, but this is subject to
// change. This function will return EOVFLOW if more bytes are being received
// than the length of rx.
//
// This function is similar to (*Device).InitiatorTransceiveBytes() with the
// following differences:
//
//  - A precise cycles counter will indicate the number of cycles between emission & reception of frames.
//  - Only modes with EASY_FRAMING option disabled are supported.
//  - Overall communication with the host is heavier and slower.
//
// By default, the timer configuration tries to maximize the precision, which
// also limits the maximum cycle count before saturation / timeout. E.g. with
// PN53x it can count up to 65535 cycles, avout 4.8ms with a precision of about
// 73ns. If you're ok with the defaults, call this function with cycles = 0. If
// you need to count more cycles, set cycles to the maximum you exprect, but
// don't forget you'll loose in precision and it'll take more time before
// timeout, so don't abuse!
//
// Warning: The configuration option EASY_FRAMING must be set to false; the
// configuration option HANDLE_PARITY must be set to true (default value).
func (d *Device) InitiatorTransceiveBytesTimed(tx, rx []byte, cycles uint32) (n int, c uint32, err error) {
	if d.d == nil {
		return ESOFT, 0, errors.New("Device closed")
	}

	var cptr *C.uint32_t
	*cptr = C.uint32_t(cycles)

	txptr := (*C.uint8_t)(&tx[0])
	rxptr := (*C.uint8_t)(&rx[0])

	n = int(C.nfc_initiator_transceive_bytes_timed(
		d.d,
		txptr, C.size_t(len(tx)),
		rxptr, C.size_t(len(rx)),
		cptr,
	))

	if n < 0 {
		err = Error(n)
	}

	c = uint32(*cptr)

	return
}

// Transceive raw bit-frames to a target. n contains the received byte count on
// success, or is meaningless on error. c will contain the actual number of
// cycles waited. The current implementation will return the libnfc error code
// in case of error, but this is subject to change. If txLength is longer than
// the supplied slice, an error will occur. txPar has to have the same length as
// tx, dito for rxPar and rx. An error will occur if any of these invariants do
// not hold.
//
// This function is similar to (*Device).InitiatorTransceiveBits() with the
// following differences:
//
//  - A precise cycles counter will indicate the number of cycles between emission & reception of frames.
//  - Only modes with EASY_FRAMING option disabled are supported and CRC must be handled manually.
//  - Overall communication with the host is heavier and slower.
//
// By default the timer configuration tries to maximize the precision, which
// also limits the maximum cycle count before saturation / timeout. E.g. with
// PN53x it can count up to 65535 cycles, avout 4.8ms with a precision of about
// 73ns. If you're ok with the defaults, call this function with cycles = 0. If
// you need to count more cycles, set cycles to the maximum you exprect, but
// don't forget you'll loose in precision and it'll take more time before
// timeout, so don't abuse!
//
// Warning: The configuration option EASY_FRAMING must be set to false; the
// configuration option HANDLE_CRC must be set to false; the configuration
// option HANDLE_PARITY must be set to true (the default value).
func (d *Device) InitiatorTransceiveBitsTimed(tx, txPar []byte, txLength int, rx, rxPar []byte, cycles uint32) (n int, c uint32, err error) {
	if d.d == nil {
		return ESOFT, 0, errors.New("Device closed")
	}

	if len(tx) != len(txPar) || len(rx) != len(rxPar) {
		return ESOFT, 0, errors.New("Invariant doesn't hold")
	}

	if len(tx) < 8*txLength {
		return ESOFT, 0, errors.New("Slice shorter than specified bit count")
	}

	var cptr *C.uint32_t
	*cptr = C.uint32_t(cycles)

	txptr := (*C.uint8_t)(&tx[0])
	txparptr := (*C.uint8_t)(&txPar[0])
	rxptr := (*C.uint8_t)(&rx[0])
	rxparptr := (*C.uint8_t)(&rxPar[0])

	n = int(C.nfc_initiator_transceive_bits_timed(
		d.d,
		txptr, C.size_t(txLength), txparptr,
		rxptr, C.size_t(len(rx)), rxparptr,
		cptr,
	))

	c = uint32(*cptr)

	if n < 0 {
		err = Error(n)
	}

	return
}

// Check target presence. Returns nil on success, an error otherwise. The
// target has to be selected before you can check its presence. To run the test,
// one or more commands will be sent to the target.
func (d *Device) InitiatorTargetIsPresent(t Target) error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	ctarget := (*C.nfc_target)(unsafe.Pointer(t.Marshall()))
	defer C.free(unsafe.Pointer(ctarget))

	n := C.nfc_initiator_target_is_present(d.d, ctarget)

	if n != 0 {
		return Error(n)
	}

	return nil
}

// Close an NFC device.
func (d *Device) Close() error {
	if d.d == nil {
		// closing a closed device is a nop
		return nil
	}

	C.nfc_close(d.d)
	d.d = nil

	return nil
}

// Abort current running command. Some commands (ie. TargetInit()) are blocking
// functions and will return only in particular conditions (ie. external
// initiator request). This function attempt to abort the current running
// command.
func (d *Device) AbortCommand() error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	return Error(C.nfc_abort_command(d.d))
}

// Turn NFC device in idle mode. In initiator mode, the RF field is turned off
// and the device is set to low power mode (if avaible); In target mode, the
// emulation is stoped (no target available from external initiator) and the
// device is set to low power mode (if avaible).
func (d *Device) Idle() error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	return Error(C.nfc_idle(d.d))
}

// Initialize NFC device as initiator (reader). After initialization it can be
// used to communicate to passive RFID tags and active NFC devices. The reader
// will act as initiator to communicate peer 2 peer (NFCIP) to other active NFC
// devices. The NFC device will be initialized with the following properties:
//  * CRC is handled by the device (NP_HANDLE_CRC = true)
//  * Parity is handled the device (NP_HANDLE_PARITY = true)
//  * Cryto1 cipher is disabled (NP_ACTIVATE_CRYPTO1 = false)
//  * Easy framing is enabled (NP_EASY_FRAMING = true)
//  * Auto-switching in ISO14443-4 mode is enabled (NP_AUTO_ISO14443_4 = true)
//  * Invalid frames are not accepted (NP_ACCEPT_INVALID_FRAMES = false)
//  * Multiple frames are not accepted (NP_ACCEPT_MULTIPLE_FRAMES = false)
//  * 14443-A mode is activated (NP_FORCE_ISO14443_A = true)
//  * speed is set to 106 kbps (NP_FORCE_SPEED_106 = true)
//  * Let the device try forever to find a target (NP_INFINITE_SELECT = true)
//  * RF field is shortly dropped (if it was enabled) then activated again
func (d *Device) InitiatorInit() error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	return Error(C.nfc_initiator_init(d.d))
}

// Initialize NFC device as initiator with its secure element initiator
// (reader). After initialization it can be used to communicate with the secure
// element. The RF field is deactivated in order to save power.
func (d *Device) InitiatorInitSecureElement() error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	return Error(C.nfc_initiator_init_secure_element(d.d))
}

// Select a passive or emulated tag.
func (d *Device) InitiatorSelectPassiveTarget(m Modulation, initData []byte) (Target, error) {
	if d.d == nil {
		return nil, errors.New("Device closed")
	}

	var pnt C.nfc_target

	err := Error(C.nfc_initiator_select_passive_target(
		d.d,
		C.nfc_modulation{C.nfc_modulation_type(m.Type), C.nfc_baud_rate(m.BaudRate)},
		(*C.uint8_t)(&initData[0]),
		C.size_t(len(initData)),
		&pnt))

	return unmarshallTarget(&pnt), err
}

// List passive or emulated tags. The NFC device will try to find the available
// passive tags. Some NFC devices are capable to emulate passive tags. The
// standards (ISO18092 and ECMA-340) describe the modulation that can be used
// for reader to passive communications. The chip needs to know with what kind
// of tag it is dealing with, therefore the initial modulation and speed (106,
// 212 or 424 kbps) should be supplied.
func (d *Device) InitiatorListPassiveTargets(m Modulation) ([]Target, error) {
	mod := C.nfc_modulation{
		nmt: C.nfc_modulation_type(m.Type),
		nbr: C.nfc_baud_rate(m.BaudRate),
	}

	tar := C.list_targets_wrapper(d.d, mod)
	defer C.free(unsafe.Pointer(tar.entries))
	if tar.count < 0 {
		return nil, Error(tar.count)
	}

	targets := make([]Target, tar.count)
	for i := range targets {
		targets[i] = unmarshallTarget(C.index_targets(tar.entries, C.int(i)))
	}

	return targets, nil
}

// Print information about an NFC device.
func (d *Device) Information() (string, error) {
	if d.d == nil {
		return "", errors.New("Device closed")
	}

	var ptr *C.char
	buflen := C.nfc_device_get_information_about(d.d, &ptr)

	if buflen < 0 {
		return "", Error(buflen)
	}

	// documentation for nfc_device_get_information_about says that buflen
	// contains the length of the string that is returned. Apparently, for
	// some drivers, buflen is always 0 so we disregard it.
	str := C.GoString(ptr)
	C.nfc_free(unsafe.Pointer(ptr))

	return str, nil
}

// Returns the device's connection string. If the device has been closed before,
// this function returns the empty string.
func (d *Device) Connection() string {
	if d.d == nil {
		return ""
	}

	cptr := C.nfc_device_get_connstring(d.d)
	return C.GoString(cptr)
}

// Returns the device's name. This information is not enough to uniquely
// determine the device.
func (d *Device) String() string {
	if d.d == nil {
		return ""
	}

	cptr := C.nfc_device_get_name(d.d)
	return C.GoString(cptr)
}

// Return Go code that could be used to reproduce this device.
func (d *Device) GoString() string {
	if d.d == nil {
		return "nil"
	}

	return fmt.Sprintf("nfc.Open(%q)", d.Connection())
}

// Set a device's integer-property value. Returns nil on success, otherwise an
// error. See integer constants in this package for possible properties.
func (d *Device) SetPropertyInt(property, value int) error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	err := C.nfc_device_set_property_int(d.d, C.nfc_property(property), C.int(value))

	if err != 0 {
		return Error(err)
	}

	return nil
}

// Set a device's boolean-property value. Returns nil on success, otherwise an
// error. See integer constants in this package for possible properties.
func (d *Device) SetPropertyBool(property int, value bool) error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	err := C.nfc_device_set_property_bool(d.d, C.nfc_property(property), C.bool(value))

	if err != 0 {
		return Error(err)
	}

	return nil
}

// Get supported modulations. Returns a slice of supported modulations or an
// error. Pass either TARGET or INITIATOR as mode. This function wraps
// nfc_device_get_supported_modulation()
func (d *Device) SupportedModulations(mode int) ([]int, error) {
	if d.d == nil {
		return nil, errors.New("Device closed")
	}

	// The documentation inside the libnfc is a bit unclear on how the
	// array returned through supported_mt is to be threated. The code
	// itself suggest that it points to an array of entries terminated with
	// UNDEFINED = 0.
	var mt_arr *C.nfc_modulation_type
	ret := C.nfc_device_get_supported_modulation(d.d, C.nfc_mode(mode), &mt_arr)
	if ret != 0 {
		return nil, Error(ret)
	}

	mods := []int{}
	type mod C.nfc_modulation_type
	ptr := unsafe.Pointer(mt_arr)

	for *(*mod)(ptr) != 0 {
		mods = append(mods, int(*(*mod)(ptr)))
		ptr = unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof(*mt_arr))
	}

	return mods, nil
}

// Get suported baud rates. Returns either a slice of supported baud rates or an
// error. This function wraps nfc_device_get_supported_baud_rate().
func (d *Device) SupportedBaudRates(modulationType int) ([]int, error) {
	if d.d == nil {
		return nil, errors.New("Device closed")
	}

	// The documentation inside the libnfc is a bit unclear on how the
	// array returned through supported_mt is to be threated. The code
	// itself suggest that it points to an array of entries terminated with
	// UNDEFINED = 0.
	var br_arr *C.nfc_baud_rate
	ret := C.nfc_device_get_supported_baud_rate(
		d.d,
		C.nfc_modulation_type(modulationType),
		&br_arr,
	)
	if ret != 0 {
		return nil, Error(ret)
	}

	brs := []int{}
	type br C.nfc_baud_rate
	ptr := unsafe.Pointer(br_arr)

	for *(*br)(ptr) != 0 {
		brs = append(brs, int(*(*br)(ptr)))
		ptr = unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof(*br_arr))
	}

	return brs, nil
}
