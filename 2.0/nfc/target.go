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
// #include "marshall.h"
import "C"
import "unsafe"
import "errors"

// allocate space using C.malloc() for a C.nfc_target.
func mallocTarget() *C.nfc_target {
	targetSize := C.size_t(unsafe.Sizeof(C.nfc_target{}))
	return (*C.nfc_target)(C.malloc(targetSize))
}

// generic implementation for the String() functions of the Target interface.
// Notice that this panics when TargetString returns an error.
func tString(t Target) string {
	str, err := TargetString(t, true)

	if err != nil {
		panic(err)
	}

	return str
}

// Go wrapper for nfc_target. Since the nfc_target structure contains a union,
// we cannot directly map it to a Go type. Modulation() can be used to figure
// out what kind of modulation was used for this Target and what type an
// interface can be casted into.
//
// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
type Target interface {
	Modulation() Modulation
	Marshall() uintptr
	String() string // uses TargetString() with verbose = true
}

// Make a string from a target with proper error reporting. This is a wrapper
// around str_nfc_target.
func TargetString(t Target, verbose bool) (string, error) {
	ptr := unsafe.Pointer(t.Marshall())

	var result *C.char = nil

	length := C.str_nfc_target(&result, (*C.nfc_target)(ptr), C.bool(verbose))
	defer C.nfc_free(unsafe.Pointer(result))

	if length < 0 {
		return "", Error(length)
	}

	return C.GoStringN(result, C.int(length)), nil
}

// Make a target from a pointer to an nfc_target. If the object you pass it not
// an nfc_target, undefined behavior occurs and your program is likely to blow
// up.
func UnmarshallTarget(ptr unsafe.Pointer) Target {
	return unmarshallTarget((*C.nfc_target)(ptr))
}

// internal wrapper with C types for convenience
func unmarshallTarget(t *C.nfc_target) Target {
	switch C.getModulationType(t) {
	case ISO14443a:
		r := unmarshallISO14443aTarget(t)
		return &r
	case Jewel:
		r := unmarshallJewelTarget(t)
		return &r
	case ISO14443b:
		r := unmarshallISO14443bTarget(t)
		return &r
	case ISO14443bi:
		r := unmarshallISO14443biTarget(t)
		return &r
	case ISO14443b2sr:
		r := unmarshallISO14443b2srTarget(t)
		return &r
	case ISO14443b2ct:
		r := unmarshallISO14443b2ctTarget(t)
		return &r
	case Felica:
		r := unmarshallFelicaTarget(t)
		return &r
	case DEP:
		r := unmarshallDEPTarget(t)
		return &r
	default:
		panic(errors.New("cannot determine target type"))
	}
}

// NFC D.E.P. (Data Exchange Protocol) active/passive mode
const (
	Undefined = iota
	Passive
	Active
)

// NFC target information in D.E.P. (Data Exchange Protocol) see ISO/IEC 18092
// (NFCIP-1). DEPTarget mirrors nfc_dep_info.
type DEPTarget struct {
	NFCID3  [10]byte // NFCID3
	DID     byte     // DID
	BS      byte     // supported send-bit rate
	BR      byte     // supported receive-bit rate
	TO      byte     // timeout value
	PP      byte     // PP parameters
	GB      [48]byte // general bytes
	GBLen   int      // length of the GB field
	DepMode int      // DEP mode
	Baud    int      // Baud rate
}

func (t *DEPTarget) String() string {
	return tString(t)
}

// Type is always DEP
func (t *DEPTarget) Modulation() Modulation {
	return Modulation{DEP, t.Baud}
}

// Make a DEPTarget from an nfc_dep_info
func unmarshallDEPTarget(c *C.nfc_target) DEPTarget {
	var dt DEPTarget

	C.unmarshallDEPTarget((*C.struct_DEPTarget)(unsafe.Pointer(&dt)), c)

	return dt
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *DEPTarget) Marshall() uintptr {
	nt := mallocTarget()
	dt := (*C.struct_DEPTarget)(unsafe.Pointer(d))

	C.marshallDEPTarget(nt, dt)

	return uintptr(unsafe.Pointer(nt))
}

// NFC ISO14443A tag (MIFARE) information. ISO14443aTarget mirrors
// nfc_iso14443a_info.
type ISO14443aTarget struct {
	Atqa   [2]byte
	Sak    byte
	UIDLen int // length of the Uid field
	UID    [10]byte
	AtsLen int // length of the ATS field
	// Maximal theoretical ATS is FSD-2, FSD=256 for FSDI=8 in RATS
	Ats  [254]byte // up to 254 bytes
	Baud int       // Baud rate
}

func (t *ISO14443aTarget) String() string {
	return tString(t)
}

// Type is always ISO14443A
func (t *ISO14443aTarget) Modulation() Modulation {
	return Modulation{ISO14443a, t.Baud}
}

// Make an ISO14443aTarget from an nfc_iso14443a_info
func unmarshallISO14443aTarget(c *C.nfc_target) ISO14443aTarget {
	var it ISO14443aTarget

	C.unmarshallISO14443aTarget((*C.struct_ISO14443aTarget)(unsafe.Pointer(&it)), c)

	return it
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *ISO14443aTarget) Marshall() uintptr {
	nt := mallocTarget()
	it := (*C.struct_ISO14443aTarget)(unsafe.Pointer(d))

	C.marshallISO14443aTarget(nt, it)

	return uintptr(unsafe.Pointer(nt))
}

// NFC FeLiCa tag information
type FelicaTarget struct {
	Len     uint
	ResCode byte
	ID      [8]byte
	Pad     [8]byte
	SysCode [2]byte
	Baud    int
}

func (t *FelicaTarget) String() string {
	return tString(t)
}

// Type is always FELICA
func (t *FelicaTarget) Modulation() Modulation {
	return Modulation{Felica, t.Baud}
}

// Make an FelicaTarget from an nfc_felica_info
func unmarshallFelicaTarget(c *C.nfc_target) FelicaTarget {
	var ft FelicaTarget

	C.unmarshallFelicaTarget((*C.struct_FelicaTarget)(unsafe.Pointer(&ft)), c)

	return ft
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *FelicaTarget) Marshall() uintptr {
	nt := mallocTarget()
	ft := (*C.struct_FelicaTarget)(unsafe.Pointer(d))

	C.marshallFelicaTarget(nt, ft)

	return uintptr(unsafe.Pointer(nt))
}

// NFC ISO14443B tag information. See ISO14443-3 for more details.
type ISO14443bTarget struct {
	Pupi            [4]byte // stores PUPI contained in ATQB (Answer To reQuest of type B)
	ApplicationData [4]byte // stores Application Data contained in ATQB
	ProtocolInfo    [3]byte // stores Protocol Info contained in ATQB
	CardIdentifier  byte    // store CID (Card Identifier) attributted by PCD to the PICC
	Baud            int
}

func (t *ISO14443bTarget) String() string {
	return tString(t)
}

// Type is always ISO14443B
func (t *ISO14443bTarget) Modulation() Modulation {
	return Modulation{ISO14443b, t.Baud}
}

// Make an ISO14443bTarget from an nfc_iso14443b_info
func unmarshallISO14443bTarget(c *C.nfc_target) ISO14443bTarget {
	var it ISO14443bTarget

	C.unmarshallISO14443bTarget((*C.struct_ISO14443bTarget)(unsafe.Pointer(&it)), c)

	return it
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *ISO14443bTarget) Marshall() uintptr {
	nt := mallocTarget()
	it := (*C.struct_ISO14443bTarget)(unsafe.Pointer(d))

	C.marshallISO14443bTarget(nt, it)

	return uintptr(unsafe.Pointer(nt))
}

// NFC ISO14443B' tag information
type ISO14443biTarget struct {
	DIV    [4]byte  // 4 LSBytes of tag serial number
	VerLog byte     // Software version & type of REPGEN
	Config byte     // Config Byte, present if long REPGEN
	AtrLen int      // length of the Atr field
	Atr    [33]byte // ATR, if any. At most 33 bytes
	Baud   int
}

func (t *ISO14443biTarget) String() string {
	return tString(t)
}

// Type is always ISO14443BI
func (t *ISO14443biTarget) Modulation() Modulation {
	return Modulation{ISO14443bi, t.Baud}
}

// Make an ISO14443biTarget from an nfc_iso14443bi_info
func unmarshallISO14443biTarget(c *C.nfc_target) ISO14443biTarget {
	var it ISO14443biTarget

	C.unmarshallISO14443biTarget((*C.struct_ISO14443biTarget)(unsafe.Pointer(&it)), c)

	return it
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *ISO14443biTarget) Marshall() uintptr {
	nt := mallocTarget()
	it := (*C.struct_ISO14443biTarget)(unsafe.Pointer(d))

	C.marshallISO14443biTarget(nt, it)

	return uintptr(unsafe.Pointer(nt))
}

// NFC ISO14443-2B ST SRx tag information
type ISO14443b2srTarget struct {
	UID  [8]byte
	Baud int
}

func (t *ISO14443b2srTarget) String() string {
	return tString(t)
}

// Type is always ISO14443B2SR
func (t *ISO14443b2srTarget) Modulation() Modulation {
	return Modulation{ISO14443b2sr, t.Baud}
}

// Make an ISO14443b2srTarget from an nfc_iso14443b2sr_info
func unmarshallISO14443b2srTarget(c *C.nfc_target) ISO14443b2srTarget {
	var it ISO14443b2srTarget

	C.unmarshallISO14443b2srTarget((*C.struct_ISO14443b2srTarget)(unsafe.Pointer(&it)), c)

	return it
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *ISO14443b2srTarget) Marshall() uintptr {
	nt := mallocTarget()
	it := (*C.struct_ISO14443b2srTarget)(unsafe.Pointer(d))

	C.marshallISO14443b2srTarget(nt, it)

	return uintptr(unsafe.Pointer(nt))
}

// NFC ISO14443-2B ASK CTx tag information
type ISO14443b2ctTarget struct {
	UID      [4]byte
	ProdCode byte
	FabCode  byte
	Baud     int
}

func (t *ISO14443b2ctTarget) String() string {
	return tString(t)
}

// Type is always ISO1444B2CT
func (t *ISO14443b2ctTarget) Modulation() Modulation {
	return Modulation{ISO14443b2ct, t.Baud}
}

// Make an ISO14443b2ctTarget from an nfc_iso14443b2ct_info
func unmarshallISO14443b2ctTarget(c *C.nfc_target) ISO14443b2ctTarget {
	var it ISO14443b2ctTarget

	C.unmarshallISO14443b2ctTarget((*C.struct_ISO14443b2ctTarget)(unsafe.Pointer(&it)), c)

	return it
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *ISO14443b2ctTarget) Marshall() uintptr {
	nt := mallocTarget()
	it := (*C.struct_ISO14443b2ctTarget)(unsafe.Pointer(d))

	C.marshallISO14443b2ctTarget(nt, it)

	return uintptr(unsafe.Pointer(nt))
}

// NFC Jewel tag information
type JewelTarget struct {
	SensRes [2]byte
	ID      [4]byte
	Baud    int
}

func (t *JewelTarget) String() string {
	return tString(t)
}

// Type is always JEWEL
func (t *JewelTarget) Modulation() Modulation {
	return Modulation{Jewel, t.Baud}
}

// Make a JewelTarget from an nfc_jewel_info
func unmarshallJewelTarget(c *C.nfc_target) JewelTarget {
	var jt JewelTarget

	C.unmarshallJewelTarget((*C.struct_JewelTarget)(unsafe.Pointer(&jt)), c)

	return jt
}

// Marshall() returns a pointer to an nfc_target allocated with C.malloc() that
// contains the same data as the Target. Don't forget to C.free() the result of
// Marshall() afterwards. A runtime panic may occur if any slice referenced by a
// Target has been made larger than the maximum length mentioned in the
// respective comments.
func (d *JewelTarget) Marshall() uintptr {
	nt := mallocTarget()
	jt := (*C.struct_JewelTarget)(unsafe.Pointer(d))

	C.marshallJewelTarget(nt, jt)

	return uintptr(unsafe.Pointer(nt))
}
