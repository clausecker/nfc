/*-
 * Copyright (c) 2014, Robert Clausecker <fuzxxl@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify it
 * under the terms of the GNU Lesser General Public License as published by the
 * Free Software Foundation, version 3.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
 * FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
 * more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 */
#ifndef MARSHALL_H
#define MARSHALL_H

#include <stddef.h>
#include <stdint.h>
#include <nfc/nfc.h>

/*
 * Marshalling code. The code in this translation unit marshalls between the
 * nfc_xxx_info and the XxxTarget types.
 *
 * The marshalling code cannot easily be written in Go as the libnfc uses
 * #pragma pack to change the alignment of some structures from something
 * perfectly usably to something cgo cannot work with. Writing the marshalling
 * code in straight C is also not possible as there is no easy way to refer to
 * Go types from within C code.
 *
 * The following code uses a somewhat hacky approach: For each Go structure we
 * create an equal C structure and hope that they match. Then we marshall the
 * data in C and cast the pointers to corresponding Go type.
 */

/*
 * On all platforms where a Go port exists, the Go type int has the same
 * attributes as the C type ptrdiff_t.
 */
typedef ptrdiff_t GoInt;

/* return field nfc_target.nm.nmt */
extern int getModulationType(const nfc_target*);

/* functions to deal with specific targets */

struct DEPTarget {
	uint8_t	NFCID3[10];
	uint8_t	DID;
	uint8_t	BS;
	uint8_t	BR;
	uint8_t	TO;
	uint8_t	PP;
	uint8_t	GB[48];
	GoInt	GBlen;
	GoInt	DepMode;
	GoInt	Baud;
};

extern void unmarshallDEPTarget(struct DEPTarget*, const nfc_target*);
extern void marshallDEPTarget(nfc_target*, const struct DEPTarget*);

struct ISO14443aTarget {
	uint8_t	Atqa[2];
	uint8_t	Sak;
	GoInt	UidLen;
	uint8_t	Uid[10];
	GoInt	AtsLen;
	uint8_t	Ats[254];
	GoInt	Baud;
};

extern void unmarshallISO14443aTarget(struct ISO14443aTarget*, const nfc_target*);
extern void marshallISO14443aTarget(nfc_target*, const struct ISO14443aTarget*);

struct FelicaTarget {
	GoInt	Len;
	uint8_t	ResCode;
	uint8_t	Id[8];
	uint8_t	Pad[8];
	uint8_t	SysCode[2];
	GoInt	Baud;
};

extern void unmarshallFelicaTarget(struct FelicaTarget*, const nfc_target*);
extern void marshallFelicaTarget(nfc_target*, const struct FelicaTarget*);

struct ISO14443bTarget {
	uint8_t	Pupi[4];
	uint8_t	ApplicationData[4];
	uint8_t	ProtocolInfo[3];
	uint8_t	CardIdentifier;
	GoInt	Baud;
};

extern void unmarshallISO14443bTarget(struct ISO14443bTarget*, const nfc_target*);
extern void marshallISO14443bTarget(nfc_target*, const struct ISO14443bTarget*);

struct ISO14443biTarget {
	uint8_t	DIV[4];
	uint8_t	VerLog;
	uint8_t	Config;
	GoInt	AtrLen;
	uint8_t	Atr[33];
	GoInt	Baud;
};

extern void unmarshallISO14443biTarget(struct ISO14443biTarget*, const nfc_target*);
extern void marshallISO14443biTarget(nfc_target*, const struct ISO14443biTarget*);

struct ISO14443b2srTarget {
	uint8_t	UID[8];
	GoInt	Baud;
};

extern void unmarshallISO14443b2srTarget(struct ISO14443b2srTarget*, const nfc_target*);
extern void marshallISO14443b2srTarget(nfc_target*, const struct ISO14443b2srTarget*);

struct ISO14443b2ctTarget {
	uint8_t	UID[4];
	uint8_t	ProdCode;
	uint8_t	FabCode;
	GoInt	Baud;
};

extern void unmarshallISO14443b2ctTarget(struct ISO14443b2ctTarget*, const nfc_target*);
extern void marshallISO14443b2ctTarget(nfc_target*, const struct ISO14443b2ctTarget*);

struct JewelTarget {
	uint8_t	SensRes[2];
	uint8_t	Id[4];
	GoInt	Baud;
};

extern void unmarshallJewelTarget(struct JewelTarget*, const nfc_target*);
extern void marshallJewelTarget(nfc_target*, const struct JewelTarget*);

#endif /* MARSHALL_H */
