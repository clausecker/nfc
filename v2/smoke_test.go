// Copyright (c) 2024 Robert Clausecker <fuzxxl@gmail.com>
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

import "testing"

// Verify that the devices can be listed without crashing.
func TestListDevices(t *testing.T) {
	devs, err := ListDevices()
	if err != nil {
		t.Log("ListDevices() failed:", err)
	} else {
		t.Log("ListDevices():", devs)
	}
}

var testModulations []Modulation = []Modulation{
	{Type: ISO14443a, BaudRate: Nbr106},
	{Type: Felica, BaudRate: Nbr212},
	{Type: Felica, BaudRate: Nbr424},
	{Type: ISO14443b, BaudRate: Nbr106},
	{Type: ISO14443bi, BaudRate: Nbr106},
	{Type: ISO14443b2sr, BaudRate: Nbr106},
	{Type: ISO14443b2ct, BaudRate: Nbr106},
	{Type: ISO14443biClass, BaudRate: Nbr106},
	{Type: Jewel, BaudRate: Nbr106},
	{Type: Barcode, BaudRate: Nbr106},
}

// Open the first device and list all tags
func TestInitiatorListPassiveTargets(t *testing.T) {
	dev, err := Open("")
	defer dev.Close()
	if err != nil {
		t.Skip("Cannot open device:", err)
	}

	for i := range testModulations {
		targets, err := dev.InitiatorListPassiveTargets(testModulations[i])
		if err != nil {
			t.Log(dev.GoString(), ".InitiatorListPassiveTargets:", err)
			t.FailNow()
		}

		t.Log(dev.Connection(), "/", testModulations[i], ":", targets)
	}
}
