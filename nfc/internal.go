package nfc

// #cgo LDFLAGS: -lnfc
// #include <stdlib.h>
// #include <nfc/nfc.h>
import "C"
import "errors"
import "unsafe"

// NFC context
type Context struct {
	context *C.nfc_context
}

// initialize the library
func Init() (c Context, err error) {
	C.nfc_init(&c.context)

	if c.context == nil {
		err = errors.New("Cannot initialize library")
	}

	return
}

// deinitialize the library
func (c Context) Exit() {
	C.nfc_exit(c.context)
}

// NFC device
type Device struct {
	device *C.nfc_device
}

// the error returned by the last operation on d. Every function that wraps some
// functions operating on an nfc_device should call this function and return the
// result. This wraps nfc_device_get_last_error.
func (d Device) lastError() error {
	err := Error(C.nfc_device_get_last_error(d.device))

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
// 1023 characters. Call Free() when the string is no longer in use
func newConnstring(s string) (connstring, error) {
	if len(s) >= BUFSIZE_CONNSTRING {
		return connstring{nil}, errors.New("String too long for Connstring")
	}

	return connstring{C.CString(s)}, nil
}

// Free's a connstring. Do not dereference afterwards.
func (c connstring) Free() {
	C.free(unsafe.Pointer(c.ptr))
}
