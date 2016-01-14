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
*/
import "C"
import "errors"
import "unsafe"

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
func (d Device) InitiatorTransceiveBytes(tx, rx []byte, timeout int) (n int, err error) {
	if *d.d == nil {
		return ESOFT, errors.New("device closed")
	}

	txptr := (*C.uint8_t)(&tx[0])
	rxptr := (*C.uint8_t)(&rx[0])

	n = int(C.nfc_initiator_transceive_bytes(
		*d.d,
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
// Device.InitiatorTransceiveBytes(). With that function you can only
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
// better can use the Device.InitiatorTransceiveBytes() method.
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
func (d Device) InitiatorTransceiveBits(tx, txPar []byte, txLength uint, rx, rxPar []byte) (n int, err error) {
	if *d.d == nil {
		return ESOFT, errors.New("device closed")
	}

	if len(tx) != len(txPar) || len(rx) != len(rxPar) {
		return ESOFT, errors.New("invariant doesn't hold")
	}

	if uint(len(tx))*8 < txLength {
		return ESOFT, errors.New("slice shorter than specified bit count")
	}

	txptr := (*C.uint8_t)(&tx[0])
	txparptr := (*C.uint8_t)(&txPar[0])
	rxptr := (*C.uint8_t)(&rx[0])
	rxparptr := (*C.uint8_t)(&rxPar[0])

	n = int(C.nfc_initiator_transceive_bits(
		*d.d,
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
// This function is similar to Device.InitiatorTransceiveBytes() with the
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
func (d Device) InitiatorTransceiveBytesTimed(tx, rx []byte, cycles uint32) (n int, c uint32, err error) {
	if *d.d == nil {
		return ESOFT, 0, errors.New("device closed")
	}

	var cptr *C.uint32_t
	*cptr = C.uint32_t(cycles)

	txptr := (*C.uint8_t)(&tx[0])
	rxptr := (*C.uint8_t)(&rx[0])

	n = int(C.nfc_initiator_transceive_bytes_timed(
		*d.d,
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
// This function is similar to Device.InitiatorTransceiveBits() with the
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
func (d Device) InitiatorTransceiveBitsTimed(tx, txPar []byte, txLength uint, rx, rxPar []byte, cycles uint32) (n int, c uint32, err error) {
	if *d.d == nil {
		return ESOFT, 0, errors.New("device closed")
	}

	if len(tx) != len(txPar) || len(rx) != len(rxPar) {
		return ESOFT, 0, errors.New("invariant doesn't hold")
	}

	if uint(len(tx))*8 < txLength {
		return ESOFT, 0, errors.New("slice shorter than specified bit count")
	}

	var cptr *C.uint32_t
	*cptr = C.uint32_t(cycles)

	txptr := (*C.uint8_t)(&tx[0])
	txparptr := (*C.uint8_t)(&txPar[0])
	rxptr := (*C.uint8_t)(&rx[0])
	rxparptr := (*C.uint8_t)(&rxPar[0])

	n = int(C.nfc_initiator_transceive_bits_timed(
		*d.d,
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
func (d Device) InitiatorTargetIsPresent(t Target) error {
	if *d.d == nil {
		return errors.New("device closed")
	}

	ctarget := (*C.nfc_target)(unsafe.Pointer(t.Marshall()))
	defer C.free(unsafe.Pointer(ctarget))

	n := C.nfc_initiator_target_is_present(*d.d, ctarget)
	if n != 0 {
		return Error(n)
	}

	return nil
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
func (d Device) InitiatorInit() error {
	if *d.d == nil {
		return errors.New("device closed")
	}

	n := C.nfc_initiator_init(*d.d)
	if n != 0 {
		return Error(n)
	}

	return nil
}

// Initialize NFC device as initiator with its secure element initiator
// (reader). After initialization it can be used to communicate with the secure
// element. The RF field is deactivated in order to save power.
func (d Device) InitiatorInitSecureElement() error {
	if *d.d == nil {
		return errors.New("device closed")
	}

	return Error(C.nfc_initiator_init_secure_element(*d.d))
}

// Select a passive or emulated tag. initData is used with different kind of
// data depending on modulation type:
//  * for an ISO/IEC 14443 type A modulation, initData contains the UID you want to select;
//  * for an ISO/IEC 14443 type B modulation, initData contains Application Family Identifier (AFI) (see ISO/IEC 14443-3) and optionally a second byte = 0x01 if you want to use probabilistic approach instead of timeslot approach;
//  * for a FeliCa modulation, initData contains a 5-byte polling payload (see ISO/IEC 18092 11.2.2.5).
//  * for ISO14443B', ASK CTx and ST SRx, see corresponding standards
// if nil, default values adequate for the chosen modulation will be used.
func (d Device) InitiatorSelectPassiveTarget(m Modulation, initData []byte) (Target, error) {
	if *d.d == nil {
		return nil, errors.New("device closed")
	}

	var pnt C.nfc_target
	var initDataPtr *C.uint8_t = nil
	var initDataLen C.size_t = 0

	if (initData != nil) {
		initDataPtr = (*C.uint8_t)(&initData[0])
		initDataLen = C.size_t(len(initData))
	}

	n := C.nfc_initiator_select_passive_target(
		*d.d,
		C.nfc_modulation{C.nfc_modulation_type(m.Type), C.nfc_baud_rate(m.BaudRate)},
		initDataPtr, initDataLen, &pnt)
	if n < 0 {
		return nil, Error(n)
	}

	return unmarshallTarget(&pnt), nil
}

// List passive or emulated tags. The NFC device will try to find the available
// passive tags. Some NFC devices are capable to emulate passive tags. The
// standards (ISO18092 and ECMA-340) describe the modulation that can be used
// for reader to passive communications. The chip needs to know with what kind
// of tag it is dealing with, therefore the initial modulation and speed (106,
// 212 or 424 kbps) should be supplied.
func (d Device) InitiatorListPassiveTargets(m Modulation) ([]Target, error) {
	if *d.d == nil {
		return nil, errors.New("device closed")
	}

	mod := C.nfc_modulation{
		nmt: C.nfc_modulation_type(m.Type),
		nbr: C.nfc_baud_rate(m.BaudRate),
	}

	tar := C.list_targets_wrapper(*d.d, mod)
	defer C.free(unsafe.Pointer(tar.entries))
	if tar.count < 0 {
		return nil, Error(tar.count)
	}

	targets := make([]Target, tar.count)
	for i := range targets {
		// index the C array using pointer arithmetic
		ptr := uintptr(unsafe.Pointer(tar.entries)) + uintptr(i)*unsafe.Sizeof(*tar.entries)
		targets[i] = unmarshallTarget((*C.nfc_target)(unsafe.Pointer(ptr)))
	}

	return targets, nil
}

// Deselect a selected passive or emulated tag. After selecting and
// communicating with a passive tag, this function could be used to deactivate
// and release the tag. This is very useful when there are multiple tags
// available in the field. It is possible to use the
// InitiatorSelectPassiveTarget() method to select the first available tag, test
// it for the available features and support, deselect it and skip to the next
// tag until the correct tag is found.
func (d Device) InitiatorDeselectTarget() error {
	if *d.d == nil {
		return errors.New("device closed")
	}

	n := C.nfc_initiator_deselect_target(*d.d)
	if n != 0 {
		return Error(n)
	}

	return nil
}
