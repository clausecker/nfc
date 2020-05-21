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

// The code in this file is a very direct translation of the code in
// iso14443-subr.c of the libnfc version 1.7.0 translated to Go for performance
// reasons.

package nfc

// Calculate an ISO 14443a CRC. Code translated from the code in
// iso14443a_crc().
func ISO14443aCRC(data []byte) [2]byte {
	crc := uint32(0x6363)
	for _, bt := range data {
		bt ^= uint8(crc & 0xff)
		bt ^= bt << 4
		bt32 := uint32(bt)
		crc = (crc >> 8) ^ (bt32 << 8) ^ (bt32 << 3) ^ (bt32 >> 4)
	}

	return [2]byte{byte(crc & 0xff), byte((crc >> 8) & 0xff)}
}

// Calculate an ISO 14443a CRC and append it to the supplied slice.
func AppendISO14443aCRC(data []byte) []byte {
	crc := ISO14443aCRC(data)
	return append(data, crc[0], crc[1])
}

// Calculate an ISO 14443b CRC. Code translated from the code in
// iso14443b_crc().
func ISO14443bCRC(data []byte) [2]byte {
	crc := uint32(0xffff)
	for _, bt := range data {
		bt ^= uint8(crc & 0xff)
		bt ^= bt << 4
		bt32 := uint32(bt)
		crc = (crc >> 8) ^ (bt32 << 8) ^ (bt32 << 3) ^ (bt32 >> 4)
	}

	return [2]byte{byte(crc & 0xff), byte((crc >> 8) & 0xff)}
}

// Calculate an ISO 14443b CRC and append it to the supplied slice.
func AppendISO14443bCRC(data []byte) []byte {
	crc := ISO14443bCRC(data)
	return append(data, crc[0], crc[1])
}

// Locate historical bytes according to ISO/IEC 14443-4 sec. 5.2.7. Return nil
// if that fails.
func ISO14443aLocateHistoricalBytes(ats []byte) []byte {
	if len(ats) > 0 {
		offset := 1
		if ats[0]&0x10 != 0 {
			offset++
		}

		if ats[1]&0x20 != 0 {
			offset++
		}

		if ats[2]&0x40 != 0 {
			offset++
		}

		if len(ats) > offset {
			return ats[offset:]
		}
	}

	return nil
}

// Add cascade tags (0x88) in UID. See ISO/IEC 14443-3 sec. 6.4.4.
func ISO14443CascadeUID(uid []byte) (cascadedUID []byte) {
	switch len(uid) {
	case 7:
		cascadedUID = make([]byte, 8)
		cascadedUID[0] = 0x88
		copy(cascadedUID[1:], uid)
	case 10:
		cascadedUID = make([]byte, 12)
		cascadedUID[0] = 0x88
		copy(cascadedUID[1:4], uid)
		cascadedUID[4] = 0x88
		copy(cascadedUID[5:], uid[3:])
	case 4:
		fallthrough
	default:
		cascadedUID = append(cascadedUID, uid...)
	}

	return
}
