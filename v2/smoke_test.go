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
