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
import "C"
import "unsafe"
import "errors"

// make a slice with bytes from buf and length l that does not overlay over buf
func byteSliceFromC(buf *C.uint8_t, l C.size_t) []byte {
	return append([]byte(nil), C.GoBytes(unsafe.Pointer(buf), C.int(l))...)
}

// make an nfc_target with modulation set. This returns a pointer allocated by
// C.malloc which needs to be free'd later on.
func makeTarget(mod, baud int) unsafe.Pointer {
	ptr := C.malloc(C.size_t(unsafe.Sizeof(C.nfc_target{})))
	(*C.nfc_target)(ptr).nm = C.nfc_modulation{
		nmt: C.nfc_modulation_type(mod),
		nbr: C.nfc_baud_rate(baud),
	}

	// C89 says: A pointer to a struct is equal to a point to its first
	// member and a pointer to a union is equal to a pointer to any of its
	// members. Therefore we can simply return ptr to fit all use cases
	return ptr
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

// internal wrapper with C types for conveinience
func unmarshallTarget(t *C.nfc_target) Target {
	ptr := unsafe.Pointer(t)
	m := Modulation{Type: int(t.nm.nmt), BaudRate: int(t.nm.nbr)}

	switch m.Type {
	case ISO14443a:
		r := unmarshallISO14443aTarget((*C.nfc_iso14443a_info)(ptr), m)
		return &r
	case Jewel:
		r := unmarshallJewelTarget((*C.nfc_jewel_info)(ptr), m)
		return &r
	case ISO14443b:
		r := unmarshallISO14443bTarget((*C.nfc_iso14443b_info)(ptr), m)
		return &r
	case ISO14443bi:
		r := unmarshallISO14443biTarget((*C.nfc_iso14443bi_info)(ptr), m)
		return &r
	case ISO14443b2sr:
		r := unmarshallISO14443b2srTarget((*C.nfc_iso14443b2sr_info)(ptr), m)
		return &r
	case ISO14443B2CT:
		r := unmarshallISO14443b2ctTarget((*C.nfc_iso14443b2ct_info)(ptr), m)
		return &r
	case Felica:
		r := unmarshallFelicaTarget((*C.nfc_felica_info)(ptr), m)
		return &r
	case DEP:
		r := unmarshallDEPTarget((*C.nfc_dep_info)(ptr), m)
		return &r
	default:
		panic(errors.New("Cannot determine target type"))
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
	GB      []byte   // general bytes, up to 48 bytes
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
func unmarshallDEPTarget(c *C.nfc_dep_info, m Modulation) DEPTarget {
	var nfcid3 [10]byte
	for i := range nfcid3 {
		nfcid3[i] = byte(c.abtNFCID3[i])
	}

	return DEPTarget{
		NFCID3: nfcid3,
		DID:    byte(c.btDID),
		BS:     byte(c.btBS),
		BR:     byte(c.btBR),
		TO:     byte(c.btTO),
		PP:     byte(c.btPP),
		GB:     byteSliceFromC(&c.abtGB[0], c.szGB),
		Baud:   m.BaudRate,
	}
}

// See documentation in Target for more details.
func (d *DEPTarget) Marshall() uintptr {
	c := (*C.nfc_dep_info)(makeTarget(DEP, d.Baud))

	for i, b := range d.NFCID3 {
		c.abtNFCID3[i] = C.uint8_t(b)
	}

	c.btDID = C.uint8_t(d.DID)
	c.btBS = C.uint8_t(d.BS)
	c.btBR = C.uint8_t(d.BR)
	c.btTO = C.uint8_t(d.TO)
	c.btPP = C.uint8_t(d.PP)

	c.szGB = C.size_t(len(d.GB))
	for i, b := range d.GB {
		c.abtGB[i] = C.uint8_t(b)
	}

	return uintptr(unsafe.Pointer(c))
}

// NFC ISO14443A tag (MIFARE) information. ISO14443aTarget mirrors
// nfc_iso14443a_info.
type ISO14443aTarget struct {
	Atqa [2]byte
	Sak  byte
	Uid  []byte // up to 10 bytes
	// Maximal theoretical ATS is FSD-2, FSD=256 for FSDI=8 in RATS
	Ats  []byte // up to 254 bytes
	Baud int    // Baud rate
}

func (t *ISO14443aTarget) String() string {
	return tString(t)
}

// Type is always ISO14443A
func (t *ISO14443aTarget) Modulation() Modulation {
	return Modulation{ISO14443a, t.Baud}
}

// Make an ISO14443aTarget from an nfc_iso14443a_info
func unmarshallISO14443aTarget(c *C.nfc_iso14443a_info, m Modulation) ISO14443aTarget {
	atqa := [2]byte{byte(c.abtAtqa[0]), byte(c.abtAtqa[1])}

	return ISO14443aTarget{
		Atqa: atqa,
		Sak:  byte(c.btSak),
		Uid:  byteSliceFromC(&c.abtUid[0], c.szUidLen),
		Ats:  byteSliceFromC(&c.abtAts[0], c.szAtsLen),
		Baud: m.BaudRate,
	}
}

// See documentation in Target for more details.
func (d *ISO14443aTarget) Marshall() uintptr {
	c := (*C.nfc_iso14443a_info)(makeTarget(ISO14443a, d.Baud))

	c.abtAtqa[0] = C.uint8_t(d.Atqa[0])
	c.abtAtqa[1] = C.uint8_t(d.Atqa[1])
	c.btSak = C.uint8_t(d.Sak)

	c.szUidLen = C.size_t(len(d.Uid))
	for i, b := range d.Uid {
		c.abtUid[i] = C.uint8_t(b)
	}

	c.szAtsLen = C.size_t(len(d.Ats))
	for i, b := range d.Ats {
		c.abtAts[i] = C.uint8_t(b)
	}

	return uintptr(unsafe.Pointer(c))
}

// NFC FeLiCa tag information
type FelicaTarget struct {
	Len     uint
	ResCode byte
	Id      [8]byte
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
func unmarshallFelicaTarget(c *C.nfc_felica_info, m Modulation) FelicaTarget {
	t := FelicaTarget{
		Len:     uint(c.szLen),
		ResCode: byte(c.btResCode),
		Baud:    m.BaudRate,
	}

	for i, b := range c.abtId {
		t.Id[i] = byte(b)
	}

	for i, b := range c.abtPad {
		t.Pad[i] = byte(b)
	}

	t.SysCode[0] = byte(c.abtSysCode[0])
	t.SysCode[1] = byte(c.abtSysCode[1])

	return t
}

// See documentation in Target for more details.
func (d *FelicaTarget) Marshall() uintptr {
	c := (*C.nfc_felica_info)(makeTarget(Felica, d.Baud))

	c.szLen = C.size_t(d.Len)
	c.btResCode = C.uint8_t(d.ResCode)

	for i, b := range d.Id {
		c.abtId[i] = C.uint8_t(b)
	}

	for i, b := range d.Pad {
		c.abtPad[i] = C.uint8_t(b)
	}

	c.abtSysCode[0] = C.uint8_t(d.SysCode[0])
	c.abtSysCode[1] = C.uint8_t(d.SysCode[1])

	return uintptr(unsafe.Pointer(c))
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
func unmarshallISO14443bTarget(c *C.nfc_iso14443b_info, m Modulation) ISO14443bTarget {
	t := ISO14443bTarget{CardIdentifier: byte(c.ui8CardIdentifier), Baud: m.BaudRate}

	for i, b := range c.abtPupi {
		t.Pupi[i] = byte(b)
	}

	for i, b := range c.abtApplicationData {
		t.ApplicationData[i] = byte(b)
	}

	for i, b := range c.abtProtocolInfo {
		t.ProtocolInfo[i] = byte(b)
	}

	return t
}

// See documentation in Target for more details.
func (d *ISO14443bTarget) Marshall() uintptr {
	c := (*C.nfc_iso14443b_info)(makeTarget(ISO14443b, d.Baud))

	c.ui8CardIdentifier = C.uint8_t(d.CardIdentifier)

	for i, b := range d.Pupi {
		c.abtPupi[i] = C.uint8_t(b)
	}

	for i, b := range d.ApplicationData {
		c.abtApplicationData[i] = C.uint8_t(b)
	}

	for i, b := range d.ProtocolInfo {
		c.abtProtocolInfo[i] = C.uint8_t(b)
	}

	return uintptr(unsafe.Pointer(c))
}

// NFC ISO14443B' tag information
type ISO14443biTarget struct {
	DIV    [4]byte // 4 LSBytes of tag serial number
	VerLog byte    // Software version & type of REPGEN
	Config byte    // Config Byte, present if long REPGEN
	Atr    []byte  // ATR, if any. At most 33 bytes
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
func unmarshallISO14443biTarget(c *C.nfc_iso14443bi_info, m Modulation) ISO14443biTarget {
	t := ISO14443biTarget{
		VerLog: byte(c.btVerLog),
		Config: byte(c.btConfig),
		Atr:    byteSliceFromC(&c.abtAtr[0], c.szAtrLen),
		Baud:   m.BaudRate,
	}

	for i, b := range c.abtDIV {
		t.DIV[i] = byte(b)
	}

	return t
}

// See documentation in Target for more details.
func (d *ISO14443biTarget) Marshall() uintptr {
	c := (*C.nfc_iso14443bi_info)(makeTarget(ISO14443bi, d.Baud))

	c.btVerLog = C.uint8_t(d.VerLog)
	c.btConfig = C.uint8_t(d.Config)

	for i, b := range d.Atr {
		c.abtAtr[i] = C.uint8_t(b)
	}

	for i, b := range d.DIV {
		c.abtDIV[i] = C.uint8_t(b)
	}

	return uintptr(unsafe.Pointer(c))
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
func unmarshallISO14443b2srTarget(c *C.nfc_iso14443b2sr_info, m Modulation) ISO14443b2srTarget {
	t := ISO14443b2srTarget{Baud: m.BaudRate}

	for i, b := range c.abtUID {
		t.UID[i] = byte(b)
	}

	return t
}

// See documentation in Target for more details.
func (d *ISO14443b2srTarget) Marshall() uintptr {
	c := (*C.nfc_iso14443b2sr_info)(makeTarget(ISO14443b2sr, d.Baud))

	for i, b := range d.UID {
		c.abtUID[i] = C.uint8_t(b)
	}

	return uintptr(unsafe.Pointer(c))
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
	return Modulation{ISO14443B2CT, t.Baud}
}

// Make an ISO14443b2ctTarget from an nfc_iso14443b2ct_info
func unmarshallISO14443b2ctTarget(c *C.nfc_iso14443b2ct_info, m Modulation) ISO14443b2ctTarget {
	t := ISO14443b2ctTarget{
		ProdCode: byte(c.btProdCode),
		FabCode:  byte(c.btFabCode),
		Baud:     m.BaudRate,
	}

	for i, b := range c.abtUID {
		t.UID[i] = byte(b)
	}

	return t
}

// See documentation in Target for more details.
func (d *ISO14443b2ctTarget) Marshall() uintptr {
	c := (*C.nfc_iso14443b2ct_info)(makeTarget(ISO14443B2CT, d.Baud))

	c.btProdCode = C.uint8_t(d.ProdCode)
	c.btFabCode = C.uint8_t(d.FabCode)

	for i, b := range d.UID {
		c.abtUID[i] = C.uint8_t(b)
	}

	return uintptr(unsafe.Pointer(c))
}

// NFC Jewel tag information
type JewelTarget struct {
	SensRes [2]byte
	Id      [4]byte
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
func unmarshallJewelTarget(c *C.nfc_jewel_info, m Modulation) JewelTarget {
	t := JewelTarget{Baud: m.BaudRate}

	t.SensRes[0] = byte(c.btSensRes[0])
	t.SensRes[1] = byte(c.btSensRes[1])

	for i, b := range c.btId {
		t.Id[i] = byte(b)
	}

	return t
}

func (d *JewelTarget) Marshall() uintptr {
	c := (*C.nfc_jewel_info)(makeTarget(Jewel, d.Baud))

	c.btSensRes[0] = C.uint8_t(d.SensRes[0])
	c.btSensRes[1] = C.uint8_t(d.SensRes[1])

	for i, b := range d.Id {
		c.btId[i] = C.uint8_t(b)
	}

	return uintptr(unsafe.Pointer(c))
}
