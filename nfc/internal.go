package nfc

/*
#cgo LDFLAGS: -lnfc
#include <stdlib.h>
#include <nfc/nfc.h>

struct device_listing {
	int count;
	char *entries;

};

struct device_listing list_devices_wrapper(nfc_context *context) {
	size_t cstr_len = 16, actual_count; // 16
	nfc_connstring *cstr = NULL;
	struct device_listing dev;

	// call nfc_list_devices as long as our array might be too short
	for (;;) {
		cstr = realloc(cstr, cstr_len * sizeof *cstr);

		actual_count = nfc_list_devices(context, cstr, cstr_len);

		if (actual_count < cstr_len) break;

		cstr_len += 16;
	}

	dev.count = actual_count;
	dev.entries = (char*)cstr;

	return dev;
}
*/
import "C"
import "errors"
import "unsafe"
import "sync"
import "fmt"

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
func ListDevices() []string {
	return theContext.listDevices()
}

// See ListDevices() for documentation
func (c *context) listDevices() []string {
	c.m.Lock()
	defer c.m.Unlock()
	c.initContext()

	dev := C.list_devices_wrapper(c.c)
	dev_entries := C.GoBytes(unsafe.Pointer(dev.entries), dev.count*BUFSIZE_CONNSTRING)
	devices := make([]string, dev.count)

	for i := range devices {
		charptr := (*C.char)(unsafe.Pointer(&dev_entries[i*BUFSIZE_CONNSTRING]))
		devices[i] = connstring{charptr}.String()
	}

	C.free(unsafe.Pointer(dev.entries))

	return devices
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
// element. The RF field is deactivated in order to save power
func (d *Device) InitiatorInitSecureElement() error {
	if d.d == nil {
		return errors.New("Device closed")
	}

	return Error(C.nfc_initiator_init_secure_element(d.d))
}

// Select a passive or emulated tag.
func (d *Device) InitiatorSelectPassiveTarget(m Modulation, initData []byte) (*Target, error) {
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

	// TODO: convert pnt to a Target

	return nil, err
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
