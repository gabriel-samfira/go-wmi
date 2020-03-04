package vm

import (
	"fmt"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"
)

// Vnic represents a virtual NIC attached to a VM
type Vnic struct {
	mgr      *Manager
	path     string
	vmPath   string
	resource *wmi.Result
}

// isPlugged returns a boolean indicating whether or not this NIC
// is plugged into a VMswitch. It may be worth returning thr actual
// VMSwitch.
func (v *Vnic) isPlugged() (bool, error) {
	return false, nil
}

// Plug will connect this NIC to a VMSwitch
func (v *Vnic) Plug(vmSwitch string) error {
	plugged, err := v.isPlugged()
	if err != nil {
		return errors.Wrap(err, "isPlugged")
	}

	if plugged {
		return fmt.Errorf("NIC already plugged into a VM switch")
	}
	return nil
}

// Unplug will disconnect this VNIC from the switch it
// is connected to.
func (v *Vnic) Unplug() error {
	plugged, err := v.isPlugged()
	if err != nil {
		return errors.Wrap(err, "isPlugged")
	}

	if plugged {
		return nil
	}
	return nil
}

// SetAccessVLAN will set the NIC in mode ACCESS with the specified VLAN ID
// on the switchport this NIC is connected to.
func (v *Vnic) SetAccessVLAN(vlanID int) error {
	plugged, err := v.isPlugged()
	if err != nil {
		return errors.Wrap(err, "isPlugged")
	}

	if plugged == false {
		return fmt.Errorf("NIC is not plugged into a VM switch")
	}
	return nil
}

// SetModetrunk will set this NIC in mode trunk, with the specified
// native VLAN ID and allowed trunk VLAN IDs
func (v *Vnic) SetModetrunk(trunkIDs []int, nativeID int) error {
	plugged, err := v.isPlugged()
	if err != nil {
		return errors.Wrap(err, "isPlugged")
	}

	if plugged == false {
		return fmt.Errorf("NIC is not plugged into a VM switch")
	}
	return nil
}
