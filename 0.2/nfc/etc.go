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
#cgo LDFLAGS: -lnfc
#include <stdlib.h>
#include <nfc/nfc.h>

struct device_listing {
	int count; // is an error code if negative
	char *entries;

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
*/
import "C"
import "errors"
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
// if the library is already initialized. This function panics if the library
// cannot be initialized for any reason.
func (c *context) initContext() {
	if c.c != nil {
		return
	}

	C.nfc_init(&c.c)

	if c.c == nil {
		panic(errors.New("Cannot initialize libnfc"))
	}

	return
}

// deinitialize the library
func (c *context) deinitContext() {
	c.m.Lock()
	defer c.m.Unlock()

	if c.c != nil {
		C.nfc_exit(c.c)
	}
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
