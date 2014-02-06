package nfc

// #cgo LDFLAGS: -lnfc
// #include <stdlib.h>
// #include <nfc/nfc.h>
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
func (c *context) open(conn string) (d Device, err error) {
	c.m.Lock()
	defer c.m.Unlock()
	c.initContext()

	cs, err := newConnstring(conn)
	if err != nil {
		return
	}

	defer cs.Free()

	d.d = C.nfc_open(c.c, cs.ptr)

	if d.d == nil {
		err = errors.New("Cannot open NFC device")
	}

	return
}

// NFC device
type Device struct {
	d *C.nfc_device
}

// the error returned by the last operation on d. Every function that wraps some
// functions operating on an nfc_device should call this function and return the
// result. This wraps nfc_device_get_last_error.
func (d Device) lastError() error {
	err := Error(C.nfc_device_get_last_error(d.d))

	if err == 0 {
		return nil
	}

	return err
}

// Connection string
type connstring struct {
	ptr *C.char
}

func (c connstring) String() string {
	return C.GoStringN(c.ptr, BUFSIZE_CONNSTRING)
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
