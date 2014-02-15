package nfc

// #include <nfc/nfc.h>
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
