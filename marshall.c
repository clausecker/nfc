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
#include <string.h>
#include "marshall.h"

/* see marshall.h for what this code does */

extern int
getModulationType(const nfc_target *nt)
{
	return (nt->nm.nmt);
}

extern void
unmarshallDEPTarget(struct DEPTarget *dt, const nfc_target *nt)
{
	const nfc_dep_info *di = &nt->nti.ndi;

	memcpy(dt->NFCID3, di->abtNFCID3, sizeof(dt->NFCID3));
	dt->DID = di->btDID;
	dt->BS = di->btBS;
	dt->BR = di->btBR;
	dt->TO = di->btTO;
	dt->PP = di->btPP;
	memcpy(dt->GB, di->abtGB, sizeof(dt->GB));
	dt->GBLen = di->szGB;
	dt->DepMode = di->ndm;

	dt->Baud = nt->nm.nbr;
}

extern void
marshallDEPTarget(nfc_target *nt, const struct DEPTarget *dt)
{
	nfc_dep_info *di = &nt->nti.ndi;

	memcpy(di->abtNFCID3, dt->NFCID3, sizeof(dt->NFCID3));
	di->btDID = dt->DID;
	di->btBS = dt->BS;
	di->btBR = dt->BR;
	di->btTO = dt->TO;
	di->btPP = dt->PP;
	memcpy(di->abtGB, dt->GB, sizeof(dt->GB));
	di->szGB = dt->GBLen;
	di->ndm = dt->DepMode;

	nt->nm.nbr = dt->Baud;
	nt->nm.nmt = NMT_DEP;
}

extern void
unmarshallISO14443aTarget(struct ISO14443aTarget *it, const nfc_target *nt)
{
	const nfc_iso14443a_info *ii = &nt->nti.nai;

	memcpy(it->Atqa, ii->abtAtqa, sizeof(it->Atqa));
	it->Sak = ii->btSak;
	it->UIDLen = ii->szUidLen;
	memcpy(it->UID, ii->abtUid, sizeof(it->UID));
	it->AtsLen = ii->szAtsLen;
	memcpy(it->Ats, ii->abtAts, sizeof(it->Ats));

	it->Baud = nt->nm.nbr;
}

extern void
marshallISO14443aTarget(nfc_target *nt, const struct ISO14443aTarget *it)
{
	nfc_iso14443a_info *ii = &nt->nti.nai;

	memcpy(ii->abtAtqa, it->Atqa, sizeof(it->Atqa));
	ii->btSak = it->Sak;
	ii->szUidLen = it->UIDLen;
	memcpy(ii->abtUid, it->UID, sizeof(it->UID));
	ii->szAtsLen = it->AtsLen;
	memcpy(ii->abtAts, it->Ats, sizeof(it->Ats));

	nt->nm.nbr = it->Baud;
	nt->nm.nmt = NMT_ISO14443A;
}

extern void
unmarshallFelicaTarget(struct FelicaTarget *ft, const nfc_target *nt)
{
	const nfc_felica_info *fi = &nt->nti.nfi;

	ft->Len = fi->szLen;
	ft->ResCode = fi->btResCode;
	memcpy(ft->ID, fi->abtId, sizeof(ft->ID));
	memcpy(ft->Pad, fi->abtPad, sizeof(ft->Pad));
	memcpy(ft->SysCode, fi->abtSysCode, sizeof(ft->SysCode));

	ft->Baud = nt->nm.nbr;
}

extern void
marshallFelicaTarget(nfc_target *nt, const struct FelicaTarget *ft)
{
	nfc_felica_info *fi = &nt->nti.nfi;

	fi->szLen = ft->Len;
	fi->btResCode = ft->ResCode;
	memcpy(fi->abtId, ft->ID, sizeof(ft->ID));
	memcpy(fi->abtPad, ft->Pad, sizeof(ft->Pad));
	memcpy(fi->abtSysCode, ft->SysCode, sizeof(ft->SysCode));

	nt->nm.nbr = ft->Baud;
	nt->nm.nmt = NMT_FELICA;
}

extern void
unmarshallISO14443bTarget(struct ISO14443bTarget *it, const nfc_target *nt)
{
	const nfc_iso14443b_info *ii = &nt->nti.nbi;

	memcpy(it->Pupi, ii->abtPupi, sizeof(it->Pupi));
	memcpy(it->ApplicationData, ii->abtApplicationData, sizeof(it->ApplicationData));
	memcpy(it->ProtocolInfo, ii->abtProtocolInfo, sizeof(it->ProtocolInfo));
	it->CardIdentifier = ii->ui8CardIdentifier;

	it->Baud = nt->nm.nbr;
}

extern void
marshallISO14443bTarget(nfc_target *nt, const struct ISO14443bTarget *it)
{
	nfc_iso14443b_info *ii = &nt->nti.nbi;

	memcpy(ii->abtPupi, it->Pupi, sizeof(it->Pupi));
	memcpy(ii->abtApplicationData, it->ApplicationData, sizeof(it->ApplicationData));
	memcpy(ii->abtProtocolInfo, it->ProtocolInfo, sizeof(it->ProtocolInfo));
	ii->ui8CardIdentifier = it->CardIdentifier;

	nt->nm.nbr = it->Baud;
	nt->nm.nmt = NMT_ISO14443B;
}

extern void
unmarshallISO14443biTarget(struct ISO14443biTarget *it, const nfc_target *nt)
{
	const nfc_iso14443bi_info *ii = &nt->nti.nii;

	memcpy(it->DIV, ii->abtDIV, sizeof(it->DIV));
	it->VerLog = ii->btVerLog;
	it->Config = ii->btConfig;
	it->AtrLen = ii->szAtrLen;
	memcpy(it->Atr, ii->abtAtr, sizeof(it->Atr));

	it->Baud = nt->nm.nbr;
}

extern void
marshallISO14443biTarget(nfc_target *nt, const struct ISO14443biTarget *it)
{
	nfc_iso14443bi_info *ii = &nt->nti.nii;

	memcpy(ii->abtDIV, it->DIV, sizeof(it->DIV));
	ii->btVerLog = it->VerLog;
	ii->btConfig = it->Config;
	ii->szAtrLen = it->AtrLen;
	memcpy(ii->abtAtr, it->Atr, sizeof(it->Atr));

	nt->nm.nbr = it->Baud;
	nt->nm.nmt = NMT_ISO14443BI;
}

extern void
unmarshallISO14443b2srTarget(struct ISO14443b2srTarget *it, const nfc_target *nt)
{
	const nfc_iso14443b2sr_info *ii = &nt->nti.nsi;

	memcpy(it->UID, ii->abtUID, sizeof(it->UID));

	it->Baud = nt->nm.nbr;
}

extern void
marshallISO14443b2srTarget(nfc_target *nt, const struct ISO14443b2srTarget *it)
{
	nfc_iso14443b2sr_info *ii = &nt->nti.nsi;

	memcpy(ii->abtUID, it->UID, sizeof(it->UID));

	nt->nm.nbr = it->Baud;
	nt->nm.nmt = NMT_ISO14443B2SR;
}

extern void
unmarshallISO14443b2ctTarget(struct ISO14443b2ctTarget *it, const nfc_target *nt)
{
	const nfc_iso14443b2ct_info *ii = &nt->nti.nci;

	memcpy(it->UID, ii->abtUID, sizeof(it->UID));
	it->ProdCode = ii->btProdCode;
	it->FabCode = ii->btFabCode;

	it->Baud = nt->nm.nbr;
}

extern void
marshallISO14443b2ctTarget(nfc_target *nt, const struct ISO14443b2ctTarget *it)
{
	nfc_iso14443b2ct_info *ii = &nt->nti.nci;

	memcpy(ii->abtUID, it->UID, sizeof(it->UID));
	ii->btProdCode = it->ProdCode;
	ii->btFabCode = it->FabCode;

	nt->nm.nbr = it->Baud;
	nt->nm.nmt = NMT_ISO14443B2CT;
}

extern void
unmarshallJewelTarget(struct JewelTarget *jt, const nfc_target *nt)
{
	const nfc_jewel_info *ji = &nt->nti.nji;

	memcpy(jt->SensRes, ji->btSensRes, sizeof(jt->SensRes));
	memcpy(jt->ID, ji->btId, sizeof(jt->ID));

	jt->Baud = nt->nm.nbr;
}

extern void
marshallJewelTarget(nfc_target *nt, const struct JewelTarget *jt)
{
	nfc_jewel_info *ji = &nt->nti.nji;

	memcpy(ji->btSensRes, jt->SensRes, sizeof(jt->SensRes));
	memcpy(ji->btId, jt->ID, sizeof(jt->ID));

	nt->nm.nbr = jt->Baud;
	nt->nm.nmt = NMT_JEWEL;
}

extern void
unmarshallBarcodeTarget(struct BarcodeTarget *bt, const nfc_target *nt)
{
	const nfc_barcode_info *bi = &nt->nti.nti;

	bt->DataLen = bi->szDataLen;
	memcpy(bt->Data, bi->abtData, sizeof(bt->Data));

	bt->Baud = nt->nm.nbr;
}

extern void
marshallBarcodeTarget(nfc_target *nt, const struct BarcodeTarget *bt)
{
	nfc_barcode_info *bi = &nt->nti.nti;

	bi->szDataLen = bt->DataLen;
	memcpy(bi->abtData, bt->Data, sizeof(bt->Data));

	nt->nm.nbr = bt->Baud;
	nt->nm.nmt = NMT_BARCODE;
}
