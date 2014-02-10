package nfc

/*
#cgo LDFLAGS: -lnfc
#include <stdlib.h>
#include <nfc/nfc.h>

struct device_listing {
	int count; // is an error code if negative
	char *entries;

};

struct target_listing {
	int count; // is an error code if negative
	nfc_target *entries;
};

struct device_listing list_devices_wrapper(nfc_context *context) {
	size_t cstr_len = 16, actual_count;
	nfc_connstring *cstr = NULL, *cstr_tmp;
	struct device_listing dev;

	// call nfc_list_devices as long as our array might be too short
	for (;;) {
		cstr_tmp = realloc(cstr, cstr_len * sizeof *cstr);
		if (cstr_tmp == NULL) {
			actual_count = NFC_ESOFT;
			break;
		}

		cstr = cstr_tmp;
		actual_count = nfc_list_devices(context, cstr, cstr_len);

		// also covers the case where actual_count is an error
		if (actual_count < cstr_len) break;

		cstr_len += 16;
	}

	dev.count = actual_count;
	dev.entries = (char*)cstr;

	return dev;
}

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
import "errors"
import "fmt"
import "sync"
import "unsafe"

// Get library version. This function returns the version of the libnfc wrapped
// by this package as returned by nfc_version().
func Version() string {
	cstr := C.nfc_version()
	return C.GoString(cstr)
}

// NFC context
type context struct {
	c *C.nfc_context
	m sync.Mutex
}

// Initialize the library. This is an internal function that assumes that the
// appropriate lock is held by the surrounding function. This function is a nop
// if the library is already initialized.
func (c *context) initContext() error {
	if c.c != nil {
		return nil
	}

	C.nfc_init(&c.c)

	if c.c == nil {
		return errors.New("Cannot initialize libnfc")
	}

	return nil
}

// deinitialize the library
func (c *context) deinitContext() {
	c.m.Lock()
	defer c.m.Unlock()

	if c.c != nil {
		C.nfc_exit(c.c)
	}
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

// Scan for discoverable supported devices (ie. only available for some drivers.
// Returns a slice of strings that can be passed to Open() to open the devices
// found.
func ListDevices() ([]string, error) {
	return theContext.listDevices()
}

// See ListDevices() for documentation
func (c *context) listDevices() ([]string, error) {
	c.m.Lock()
	defer c.m.Unlock()
	c.initContext()

	dev := C.list_devices_wrapper(c.c)
	defer C.free(unsafe.Pointer(dev.entries))
	if dev.count < 0 {
		return nil, Error(dev.count)
	}

	dev_entries := C.GoBytes(unsafe.Pointer(dev.entries), dev.count*BUFSIZE_CONNSTRING)

	devices := make([]string, dev.count)
	for i := range devices {
		charptr := (*C.char)(unsafe.Pointer(&dev_entries[i*BUFSIZE_CONNSTRING]))
		devices[i] = connstring{charptr}.String()
	}

	return devices, nil
}

// NFC device. Copying these structs may cause unintended side effects.
type Device struct {
	d *C.nfc_device
}

// Return a pointer to the wrapped nfc_device. This is useful if you try to use
// this wrapper to wrap other C code that builds onto the libnfc.
func (d *Device) Pointer() uintptr {
	return uintptr(unsafe.Pointer(d.d))
}

// the error returned by the last operation on d. Every function that wraps some
// functions operating on an nfc_device should call this function and return the
// result. This wraps nfc_device_get_last_error.
func (d *Device) lastError() error {
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

// Connection string
type connstring struct {
	ptr *C.char
}

func (c connstring) String() string {
	str := C.GoStringN(c.ptr, BUFSIZE_CONNSTRING)
	i := 0

	for ; i < len(str) && str[i] != '\000'; i++ {
	}

	return str[:i+1]
}

// Makes a connstring. Notice that the string must not be longer than
// 1023 characters. If "" is passed, return nil instead. Call Free() when the
// string is no longer in use.
func newConnstring(s string) (connstring, error) {
	if s == "" {
		return connstring{nil}, nil
	}

	if len(s) >= BUFSIZE_CONNSTRING {
		return connstring{nil}, errors.New("String too long for Connstring")
	}

	return connstring{C.CString(s)}, nil
}

// Frees a connstring. Do not dereference afterwards. Free can be called on
// connstrings that contain a nil pointer.
func (c connstring) Free() {
	if c.ptr != nil {
		C.free(unsafe.Pointer(c.ptr))
	}
}
