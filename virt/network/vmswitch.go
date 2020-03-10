package network

import (
	"fmt"

	"github.com/gabriel-samfira/go-wmi/utils"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

// NewVMSwitchManager returns a new Manager type
func NewVMSwitchManager() (*Manager, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}

	// Get virtual machine management service
	svc, err := w.GetOne(VMSwitchManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}

	standardCim, err := wmi.NewConnection(".", `root\StandardCimv2`)
	if err != nil {
		return nil, err
	}

	sw := &Manager{
		con:         w,
		stdCimV2Con: standardCim,
		svc:         svc,
	}
	return sw, nil
}

// Manager offers a root\virtualization\v2 instance connection
// and an instance of Msvm_VirtualEthernetSwitchManagementService
type Manager struct {
	con         *wmi.WMI
	stdCimV2Con *wmi.WMI
	svc         *wmi.Result
}

// Release closes the WMI connection
func (m *Manager) Release() {
	m.con.Close()
	m.stdCimV2Con.Close()
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

// CreateVMSwitch will create a new private VM Switch. To convert it into an internal
// or external VMSwitch, call the SetInternalPort() and SetExternalPort() functions.
// Calling SetInternalPort() on a private switch will turn it into an "internal"
// switch. An internal switch will only facilitate traffic between VMs attached to it
// and the Host operating system. Attaching an internal port means the OS can manage
// the IP settings of the switch.
// Calling SetExternalPort() will attach an physical or virtual net adapter to the
// VM switch. This will allow trafic to flow through that interface, making it an
// external VM switch.
// If both SetInternalPort() and SetExternalPort() are called, the switch becomes an
// external VM switch with management OS, inheriting the IP settings of the external
// net adapter attached to the switch.
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

// RemoveVMSwitch will delete a VMSwitch
func (m *Manager) RemoveVMSwitch(switchID string) error {
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

func (v VirtualSwitch) getHostResourceLocation(res *wmi.Result) (*wmi.Location, error) {
	hostResource, err := res.GetProperty("HostResource")
	if err != nil {
		return nil, errors.Wrap(err, "HostResource")
	}

	asArray := hostResource.ToArray()
	valueArray := asArray.ToValueArray()
	if len(valueArray) == 0 {
		return nil, wmi.ErrNotFound
	}
	valuePath := valueArray[0].(string)
	location, err := wmi.NewLocation(valuePath)
	if err != nil {
		return nil, errors.Wrap(err, "NewLocation")
	}
	return location, nil
}

func (v VirtualSwitch) getSwitchPortAllocSettings() ([]switchPortAllocations, error) {
	portsData, err := v.activeSettingsData.Get("associators_", nil, PortAllocSetData)
	if err != nil {
		return nil, errors.Wrap(err, "associators_")
	}
	ports, err := portsData.Elements()
	if err != nil {
		return nil, errors.Wrap(err, "Elements")
	}

	ret := make([]switchPortAllocations, len(ports))
	for idx, val := range ports {
		pth, err := val.Path()
		if err != nil {
			return nil, errors.Wrap(err, "path_")
		}

		hostLocation, err := v.getHostResourceLocation(val)
		if err != nil {
			if err == wmi.ErrNotFound {
				continue
			}
			return nil, errors.Wrap(err, "getHostResourceLocation")
		}

		ret[idx] = switchPortAllocations{
			mgr:                  v.mgr,
			path:                 pth,
			hostResourceLocation: hostLocation,
			activeSettingsData:   val,
		}
	}
	return ret, nil
}

func (v VirtualSwitch) getExternalPort(deviceID string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "DeviceID",
				Value: fmt.Sprintf("Microsoft:%s", deviceID),
				Type:  wmi.Equals},
		},
	}
	fields := []string{}
	result, err := v.mgr.con.GetOne(ExternalPort, fields, qParams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (v VirtualSwitch) getHostPath() (string, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "InstallDate",
				Value: "NULL",
				Type:  wmi.Is},
		},
	}
	computerSystemresult, err := v.mgr.con.GetOne(ComputerSystem, []string{}, qParams)
	if err != nil {
		return "", errors.Wrap(err, "GetOne")
	}

	pth, err := computerSystemresult.Path()
	if err != nil {
		return "", errors.Wrap(err, "path_")
	}

	return pth, nil
}

func (v VirtualSwitch) getExternalPortAllocationText(externalPortPath string) (string, error) {
	// externalPort, err := v.getExternalPort(interfaceID)
	// if err != nil {
	// 	return "", errors.Wrap(err, "getExternalPort")
	// }

	// extPortPath, err := externalPort.Path()
	// if err != nil {
	// 	return "", fmt.Errorf("Could not call path_: %v", err)
	// }

	defaultSettings, err := utils.GetResourceAllocSettings(
		v.mgr.con, ETHConnResSubType, PortAllocSetData)
	if err != nil {
		return "", errors.Wrap(err, "GetResourceAllocSettings")
	}

	if err := defaultSettings.Set("HostResource", []string{externalPortPath}); err != nil {
		return "", errors.Wrap(err, "HostResource")
	}

	extText, err := defaultSettings.GetText(1)
	if err != nil {
		return "", errors.Wrap(err, "GetText")
	}
	return extText, nil
}

func (v VirtualSwitch) getInternalPortAllocationText(macAddress string) (string, error) {
	defaultSettings, err := utils.GetResourceAllocSettings(
		v.mgr.con, ETHConnResSubType, PortAllocSetData)
	if err != nil {
		return "", errors.Wrap(err, "GetResourceAllocSettings")
	}

	switchName, err := v.Name()
	if err != nil {
		return "", errors.Wrap(err, "Name")
	}

	hostPath, err := v.getHostPath()
	if err != nil {
		return "", errors.Wrap(err, "getHostPath")
	}
	if err := defaultSettings.Set("HostResource", []string{hostPath}); err != nil {
		return "", errors.Wrap(err, "HostResource")
	}

	if macAddress != "" {
		if err := defaultSettings.Set("Address", macAddress); err != nil {
			return "", errors.Wrap(err, "Set(Address)")
		}
	}

	if err := defaultSettings.Set("ElementName", switchName); err != nil {
		return "", errors.Wrap(err, "Set(ElementName)")
	}

	extText, err := defaultSettings.GetText(1)
	if err != nil {
		return "", errors.Wrap(err, "GetText")
	}
	return extText, nil
}

func (v VirtualSwitch) getSwitchExternalPortAllocSettings() (string, error) {
	allocSettings, err := v.getSwitchPortAllocSettings()
	if err != nil {
		return "", errors.Wrap(err, "getSwitchPortAllocSettings")
	}

	for _, val := range allocSettings {
		if val.hostResourceLocation.Class == ExternalPort {
			return val.Path(), nil
		}
	}
	return "", wmi.ErrNotFound
}

func (v VirtualSwitch) getSwitchInternalPort() (string, error) {
	allocSettings, err := v.getSwitchPortAllocSettings()
	if err != nil {
		return "", errors.Wrap(err, "getSwitchPortAllocSettings")
	}

	for _, val := range allocSettings {
		if val.hostResourceLocation.Class == ComputerSystem {
			return val.Path(), nil
		}
	}
	return "", wmi.ErrNotFound
}

func (v VirtualSwitch) setSwitchResources(resources []string) error {
	switchPath, err := v.activeSettingsData.Path()
	if err != nil {
		return errors.Wrap(err, "Path_")
	}
	jobPath := ole.VARIANT{}
	jobState, err := v.mgr.svc.Get("AddResourceSettings", switchPath, resources, nil, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call AddResourceSettings: %v", err)
	}

	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (v VirtualSwitch) removeSwitchResources(resources []string) error {
	jobPath := ole.VARIANT{}
	jobState, err := v.mgr.svc.Get("RemoveResourceSettings", resources, &jobPath)
	if err != nil {
		return errors.Wrap(err, "RemoveResourceSettings")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "WaitForJob")
		}
	}
	return nil
}

// SetExternalPort will attach an external ethernet port to this switch.
// This operation will make this VMswitch an "external" VM switch.
func (v VirtualSwitch) SetExternalPort(interfaceID string) error {
	_, err := v.getSwitchExternalPortAllocSettings()
	if err != nil {
		if err != wmi.ErrNotFound {
			return errors.Wrap(err, "getSwitchExternalPortAllocSettings")
		}
	}

	externalPort, err := v.getExternalPort(interfaceID)
	if err != nil {
		return errors.Wrap(err, "getExternalPort")
	}

	extPortPath, err := externalPort.Path()
	if err != nil {
		return fmt.Errorf("Could not call path_: %v", err)
	}

	extText, err := v.getExternalPortAllocationText(extPortPath)
	if err != nil {
		return errors.Wrap(err, "getExternalPortAllocationText")
	}
	resources := []string{
		extText,
	}

	err = v.setSwitchResources(resources)
	if err != nil {
		return errors.Wrap(err, "setSwitchResources")
	}
	return nil
}

// ClearExternalPort will attach an external ethernet port to this switch.
// This operation will make this VMswitch an "external" VM switch.
func (v VirtualSwitch) ClearExternalPort() (bool, error) {
	extPort, err := v.getSwitchExternalPortAllocSettings()
	if err != nil {
		if err != wmi.ErrNotFound {
			return false, errors.Wrap(err, "getSwitchExternalPortAllocSettings")
		}
	}
	if extPort == "" {
		return false, nil
	}

	resources := []string{
		extPort,
	}

	removed, err := v.ClearInternalPort()
	if err != nil {
		return false, errors.Wrap(err, "ClearInternalPort")
	}
	err = v.removeSwitchResources(resources)
	if err != nil {
		return false, errors.Wrap(err, "removeSwitchResources")
	}

	if removed {
		// management OS was enabled on the switch. Re-add the internal port
		err = v.SetInternalPort()
		if err != nil {
			return false, errors.Wrap(err, "SetInternalPort")
		}
	}
	return true, nil
}

// SetInternalPort will create an internal port which will allow the OS
// to manage this switches network settings.
func (v VirtualSwitch) SetInternalPort() error {
	externalPortAllocSettings, err := v.getSwitchExternalPortAllocSettings()
	if err != nil {
		if err != wmi.ErrNotFound {
			return errors.Wrap(err, "getSwitchExternalPortAllocSettings")
		}
	}
	var macAddress string
	if externalPortAllocSettings != "" {
		extAllocSettingsLocation, err := wmi.NewLocation(externalPortAllocSettings)
		if err != nil {
			return errors.Wrap(err, "NewLocation")
		}
		extAllocResult, err := extAllocSettingsLocation.GetResult()
		if err != nil {
			return errors.Wrap(err, "GetResult")
		}
		extPortResult, err := v.getHostResourceLocation(extAllocResult)
		if err != nil {
			return errors.Wrap(err, "getHostResourceLocation")
		}

		extResult, err := extPortResult.GetResult()
		if err != nil {
			return errors.Wrap(err, "GetResult")
		}

		mac, err := extResult.GetProperty("PermanentAddress")
		if err != nil {
			return errors.Wrap(err, "GetProperty(PermanentAddress)")
		}
		macAddress = mac.Value().(string)
	}

	internalPortText, err := v.getInternalPortAllocationText(macAddress)
	if err != nil {
		return errors.Wrap(err, "getInternalPortAllocationText")
	}
	resources := []string{
		internalPortText,
	}
	return v.setSwitchResources(resources)
}

// ClearInternalPort will remove the internal port from this switch, disabling
// the management OS.
func (v VirtualSwitch) ClearInternalPort() (bool, error) {
	internalPort, err := v.getSwitchInternalPort()
	if err != nil {
		if err != wmi.ErrNotFound {
			return false, errors.Wrap(err, "getSwitchInternalPort")
		}
	}

	if internalPort == "" {
		return false, nil
	}

	resources := []string{
		internalPort,
	}
	err = v.removeSwitchResources(resources)
	if err != nil {
		return false, errors.Wrap(err, "removeSwitchResources")
	}
	return true, nil
}

// Name returns the name of the switch
func (v VirtualSwitch) Name() (string, error) {
	name, err := v.virtualSwitch.GetProperty("ElementName")
	if err != nil {
		return "", errors.Wrap(err, "GetProperty(ElementName)")
	}
	return name.Value().(string), nil
}

// ID returns the ID of the switch
func (v VirtualSwitch) ID() (string, error) {
	id, err := v.virtualSwitch.GetProperty("Name")
	fmt.Println(id.Value(), err)
	if err != nil {
		return "", errors.Wrap(err, "GetProperty(Name)")
	}
	return id.Value().(string), nil
}

// SetName renames the switch.
// Note: this will not change the name of the internal port, which means
// that you will still see the old name when you run Get-NetAdapter or ipconfig
func (v VirtualSwitch) SetName(newName string) error {
	// Get fresh settings info
	switchSettingsResult, err := v.virtualSwitch.Get("associators_", nil, VMSwitchSettings)
	if err != nil {
		return errors.Wrap(err, "associators_")
	}

	result, err := switchSettingsResult.ItemAtIndex(0)
	if err != nil {
		return errors.Wrap(err, "ItemAtIndex")
	}

	if err := result.Set("ElementName", newName); err != nil {
		return errors.Wrap(err, "ElementName")
	}

	text, err := result.GetText(1)
	if err != nil {
		return errors.Wrap(err, "GetText")
	}
	jobPath := ole.VARIANT{}
	jobState, err := v.mgr.svc.Get("ModifySystemSettings", text, &jobPath)
	if err != nil {
		return errors.Wrap(err, "ModifySystemSettings")
	}

	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "WaitForJob")
		}
	}
	return nil
}

// SetNAT will enable NAT on an "internal" VM switch. This mode is not
// supported on older versions of Windows.
func (v VirtualSwitch) SetNAT(cird string) error {
	return nil
}

type switchPortAllocations struct {
	mgr *Manager

	path                 string
	hostResourceLocation *wmi.Location
	activeSettingsData   *wmi.Result
}

// Path returns the wmi path of this resource
func (s *switchPortAllocations) Path() string {
	return s.path
}
