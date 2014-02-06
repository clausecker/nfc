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

// Connection string
type connstring struct {
	ptr *C.char
}

func (c connstring) String() string {
	str := C.GoStringN(c.ptr, BUFSIZE_CONNSTRING)
	i := 0

	for ; i < len(str) && str[i] != '\000'; i++ { }

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
