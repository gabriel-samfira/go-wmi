package network

import (
	"fmt"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

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

// Release closes the WMI connection
func (m *Manager) Release() {
	m.con.Close()
}

func (m *Manager) getVMSwitchFromResult(sw *wmi.Result) (VirtualSwitch, error) {
	switchSettingsResult, err := sw.Get("associators_", nil, VMSwitchSettings)
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "get VMSwitchSettings")
	}

	elem, err := switchSettingsResult.Elements()
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "switchSettingsResult Elements")
	}

	if len(elem) == 0 {
		return VirtualSwitch{}, fmt.Errorf("failed to get switch settings")
	}

	switchSettings := elem[0]

	pth, err := sw.Path()
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "get switch Path_")
	}
	return VirtualSwitch{
		mgr: m,

		path:               pth,
		activeSettingsData: switchSettings,
		virtualSwitch:      sw,
	}, nil
}

func (m *Manager) getVirtualSwitches(qParams []wmi.Query) ([]VirtualSwitch, error) {
	fields := []string{}

	result, err := m.con.Gwmi(VMSwitchClass, fields, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "Gwmi VMSwitchClass")
	}

	switches, err := result.Elements()
	if err != nil {
		return nil, errors.Wrap(err, "fetching switches")
	}

	ret := make([]VirtualSwitch, len(switches))
	for idx, val := range switches {
		data, err := m.getVMSwitchFromResult(val)
		if err != nil {
			return nil, errors.Wrap(err, "getVMSwitchFromResult")
		}
		ret[idx] = data
	}

	return ret, nil
}

// GetVMSwitch returns a *VirtualSwitch given the ID of the switch
func (m *Manager) GetVMSwitch(switchID string) (VirtualSwitch, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "Name",
				Value: switchID,
				Type:  wmi.Equals,
			},
		},
	}

	switches, err := m.getVirtualSwitches(qParams)
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "getVirtualSwitches")
	}
	if len(switches) == 0 {
		return VirtualSwitch{}, wmi.ErrNotFound
	}

	if len(switches) != 1 {
		return VirtualSwitch{}, fmt.Errorf("got multiple switches from query. Expected 1")
	}
	return switches[0], nil
}

// GetVMSwitchByName will return a list of VM switches that have the specified
// name. VM switch names are non unique, so we return a list.
func (m *Manager) GetVMSwitchByName(name string) ([]VirtualSwitch, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "ElementName",
				Value: name,
				Type:  wmi.Equals,
			},
		},
	}

	return m.getVirtualSwitches(qParams)
}

// ListVMSwitches returns a list ov virtual switches
func (m *Manager) ListVMSwitches() ([]VirtualSwitch, error) {
	return m.getVirtualSwitches(nil)
}

// CreateVMSwitch will create a new VM Switch
func (m *Manager) CreateVMSwitch(name string) (VirtualSwitch, error) {
	data, err := m.con.Get(VMSwitchSettings)
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "get VMSwitchSettings")
	}

	swInstance, err := data.Get("SpawnInstance_")
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "SpawnInstance_")
	}

	if err := swInstance.Set("ElementName", name); err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "set ElementName")
	}
	switchText, err := swInstance.GetText(1)
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "GetText")
	}

	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := m.svc.Get("DefineSystem", switchText, nil, nil, &resultingSystem, &jobPath)
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "DefineSystem")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return VirtualSwitch{}, errors.Wrap(err, "WaitForJob")
		}
	}

	// The resultingSystem value for DefineSystem is always a string containing the
	// location of the newly created resource
	locationURI := resultingSystem.Value().(string)
	loc, err := wmi.NewLocation(locationURI)
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "getting location")
	}

	result, err := loc.GetResult()
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "getting result")
	}

	id, err := result.GetProperty("Name")
	if err != nil {
		return VirtualSwitch{}, errors.Wrap(err, "get Name")
	}

	return m.GetVMSwitch(id.Value().(string))
}

// RemoveVMSwitch will delete a VMSwitch. The force parameter is mandatory
// if this VMSwitch has ports attached
func (m *Manager) RemoveVMSwitch(switchID string, force bool) error {
	sw, err := m.GetVMSwitch(switchID)
	if err != nil {
		if err == wmi.ErrNotFound {
			return nil
		}
		return errors.Wrap(err, "GetVMSwitch")
	}
	swPapth, err := sw.Path()
	if err != nil {
		return errors.Wrap(err, "get path_")
	}
	jobPath := ole.VARIANT{}
	jobState, err := m.svc.Get("DestroySystem", swPapth, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DestroySystem: %v", err)
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "WaitForJob")
		}
	}
	return nil
}

// VirtualSwitch represents one virtual switch
type VirtualSwitch struct {
	mgr *Manager

	activeSettingsData *wmi.Result
	virtualSwitch      *wmi.Result
	path               string
}

// Path returns the WMI locator path of this switch
func (v VirtualSwitch) Path() (string, error) {
	return v.virtualSwitch.Path()
}

// SetExternalPort will attach an external ethernet port to this switch.
// This operation will make this VMswitch an "external" VM switch.
func (v VirtualSwitch) SetExternalPort(portID string) error {
	return nil
}

// ClearExternalPort will attach an external ethernet port to this switch.
// This operation will make this VMswitch an "external" VM switch.
func (v VirtualSwitch) ClearExternalPort() error {
	return nil
}

// SetNAT will enable NAT on an "internal" VM switch. This mode is not
// supported on older versions of Windows. On newer versions of Windows
// Only one NAT enabled switch can exist on a system.
func (v VirtualSwitch) SetNAT(cird string) error {
	return nil
}

// Name returns the name of the switch
func (v *VirtualSwitch) Name() (string, error) {
	return "", nil
}

// ID returns the ID of the switch
func (v VirtualSwitch) ID() (string, error) {
	return "", nil
}

// SetName renames the switch
func (v VirtualSwitch) SetName(newName string) error {
	return nil
}
