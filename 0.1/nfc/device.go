// Copyright (c) 2014, Robert Clausecker <fuzxxl@gmail.com>
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

package nfc

// #include <nfc/nfc.h>
// #include <stdlib.h>
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

// Initialize NFC device as an emulated tag. n contains the received byte count
// on success, or is meaningless on error. The current implementation will
// return the libnfc error code in case of error, but this is subject to change.
// The returned target tt will be the result of the modifications
// nfc_target_init() applies to t. Such modifications might happen if you set
// an Baud or DepMode to UNDEFINED. The fields will be updated with concrete
// values. timeout contains a timeout in milliseconds.
//
// This function initializes NFC device in target mode in order to emulate a tag
// as the specified target t.
//  - Crc is handled by the device (HANDLE_CRC = true)
//  - Parity is handled the device (HANDLE_PARITY = true)
//  - Cryto1 cipher is disabled (ACTIVATE_CRYPTO1 = false)
//  - Auto-switching in ISO14443-4 mode is enabled (AUTO_ISO14443_4 = true)
//  - Easy framing is disabled (EASY_FRAMING = false)
//  - Invalid frames are not accepted (ACCEPT_INVALID_FRAMES = false)
//  - Multiple frames are not accepted (ACCEPT_MULTIPLE_FRAMES = false)
//  - RF field is dropped
//
// Warning: Be aware that this function will wait (hang) until a command is
// received that is not part of the anti-collision. The RATS command for example
// would wake up the emulator. After this is received, the send and receive
// functions can be used.
//
// If timeout equals to 0, the function blocks indefinitely (until an error is
// raised or function is completed). If timeout equals to -1, the default
// timeout will be used.
func (d *Device) TargetInit(t Target, rx []byte, timeout int) (n int, tt Target, err error) {
	if d.d == nil {
		return ESOFT, t, errors.New("device closed")
	}

	tar := (*C.nfc_target)(unsafe.Pointer(t.Marshall()))
	defer C.free(unsafe.Pointer(tar))

	n = int(C.nfc_target_init(
		d.d, tar,
		(*C.uint8_t)(&rx[0]), C.size_t(len(rx)),
		C.int(timeout),
	))

	if n < 0 {
		err = Error(n)
	}

	tt = unmarshallTarget(tar)
	return
}

// Send bytes and APDU frames. n contains the sent byte count on success, or is
// meaningless on error. The current implementation will return the libnfc
// error code in case of error, but this is subject to change. timeout contains
// a timeout in milliseconds.
//
// This function make the NFC device (configured as  target) send byte frames
// (e.g. APDU responses) to the initiator.
//
// If timeout equals to 0, the function blocks indefinitely (until an error is
// raised or function is completed). If timeout equals to -1, the default
// timeout will be used.
func (d *Device) TargetSendBytes(tx []byte, timeout int) (n int, err error) {
	if d.d == nil {
		return ESOFT, errors.New("device closed")
	}

	n = int(C.nfc_target_send_bytes(
		d.d,
		(*C.uint8_t)(&tx[0]), C.size_t(len(tx)),
		C.int(timeout),
	))

	if n < 0 {
		err = Error(n)
	}

	return
}

// Receive bytes and APDU frames. n contains the received byte count on success,
// or is meaningless on error. The current implementation will return the libnfc
// error code in case of error, but this is subject to change. timeout contains
// a timeout in milliseconds.
//
// This function retrieves bytes frames (e.g. ADPU) sent by the initiator to the
// NFC device (configured as target).
//
// If timeout equals to 0, the function blocks indefinitely (until an error is
// raised or function is completed). If timeout equals to -1, the default
// timeout will be used.
func (d *Device) TargetReceiveBytes(rx []byte, timeout int) (n int, err error) {
	if d.d == nil {
		return ESOFT, errors.New("device closed")
	}

	n = int(C.nfc_target_receive_bytes(
		d.d,
		(*C.uint8_t)(&rx[0]), C.size_t(len(rx)),
		C.int(timeout),
	))

	if n < 0 {
		err = Error(n)
	}

	return
}

// Send raw bit-frames. Returns sent bits count on success, n contains the sent
// bit count on success, or is meaningless on error. The current implementation
// will return the libnfc error code in case of error, but this is subject to
// change. txPar has to have the same length as tx, an error will occur if this
// invariant does not hold.
//
// tx contains a byte slice of the frame that needs to be transmitted. txLength
// contains its length in bits. txPar contains a byte slice of the corresponding
// parity bits needed to send per byte.
//
// his function can be used to transmit (raw) bit-frames to the initiator using
// the specified NFC device (configured as target).
func (d *Device) TargetSendBits(tx []byte, txPar []byte, txLength uint) (n int, err error) {
	if d.d == nil {
		return ESOFT, errors.New("device closed")
	}

	if len(tx) != len(txPar) {
		return ESOFT, errors.New("Invariant doesn't hold")
	}

	if uint(len(tx))*8 < txLength {
		return ESOFT, errors.New("Slice shorter than specified bit count")
	}

	n = int(C.nfc_target_send_bits(
		d.d,
		(*C.uint8_t)(&tx[0]),
		C.size_t(txLength),
		(*C.uint8_t)(&txPar[0]),
	))

	if n < 0 {
		err = Error(n)
	}

	return
}

// Receive bit-frames. Returns received bits count on success, n contains the
// received bit count on success, or is meaningless on error. The current
// implementation will return the libnfc error code in case of error, but this
// is subject to change. rxPar has to have the same length as rx, an error will
// occur if this invariant does not hold.
//
// rx contains a byte slice of the frame that you want to receive. rxLength
// contains its length in bits. rxPar contains a byte slice of the corresponding
// parity bits received per byte.
//
// This function makes it possible to receive (raw) bit-frames. It returns all
// the messages that are stored in the FIFO buffer of the PN53x chip. It
// does not require to send any frame and thereby could be used to snoop frames
// that are transmitted by a nearby initiator. Check out the
// ACCEPT_MULTIPLE_FRAMES configuration option to avoid losing transmitted
// frames.
func (d *Device) TargetTransceiveBits(rx []byte, rxPar []byte, rxLength uint) (n int, err error) {
	if d.d == nil {
		return ESOFT, errors.New("device closed")
	}

	if len(rx) != len(rxPar) {
		return ESOFT, errors.New("Invariant doesn't hold")
	}

	if uint(len(rx))*8 < rxLength {
		return ESOFT, errors.New("Slice shorter than specified bit count")
	}

	n = int(C.nfc_target_receive_bits(
		d.d,
		(*C.uint8_t)(&rx[0]),
		C.size_t(rxLength),
		(*C.uint8_t)(&rxPar[0]),
	))

	if n < 0 {
		err = Error(n)
	}

	return
}
