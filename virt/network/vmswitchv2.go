package network

import "github.com/gabriel-samfira/go-wmi/wmi"

// NewVMManager returns a new Manager type
func NewVMManager() (*Manager, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}

	// Get virtual machine management service
	svc, err := w.GetOne(VMSwitchManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}

	sw := &Manager{
		con: w,
		svc: svc,
	}
	return sw, nil
}

// Manager offers a root\virtualization\v2 instance connection
// and an instance of Msvm_VirtualEthernetSwitchManagementService
type Manager struct {
	con *wmi.WMI
	svc *wmi.Result
}

// GetVMSwitch returns a *VirtualSwitch given the ID of the switch
func (m *Manager) GetVMSwitch(switchID string) (*VirtualSwitch, error) {
	return nil, nil
}

// ListVMSwitches returns a list ov virtual switches
func (m *Manager) ListVMSwitches() ([]VirtualSwitch, error) {
	return nil, nil
}

// CreateVMSwitch will create a new VM Switch
func (m *Manager) CreateVMSwitch(name string) (*VirtualSwitch, error) {
	return nil, nil
}

// RemoveVMSwitch will delete a VMSwitch. The force parameter is mandatory
// if this VMSwitch has ports attached
func (m *Manager) RemoveVMSwitch(switchID string, force bool) error {
	return nil
}

// VirtualSwitch represents one virtual switch
type VirtualSwitch struct {
	mgr *Manager

	activeSettingsData *wmi.Result
	virtualSwitch      *wmi.Result
	path               string
}

// SetExternalPort will attach an external ethernet port to this switch.
// This operation will make this VMswitch an "external" VM switch.
func (v *VirtualSwitch) SetExternalPort(portID string) error {
	return nil
}

// ClearExternalPort will attach an external ethernet port to this switch.
// This operation will make this VMswitch an "external" VM switch.
func (v *VirtualSwitch) ClearExternalPort() error {
	return nil
}

// SetNAT will enable NAT on an "internal" VM switch. This mode is not
// supported on older versions of Windows. On newer versions of Windows
// Only one NAT enabled switch can exist on a system.
func (v *VirtualSwitch) SetNAT(cird string) error {
	return nil
}
