package network

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-ole/go-ole"

	"github.com/gabriel-samfira/go-wmi/wmi"
)

var mutex = sync.RWMutex{}

type vmswitchPorts struct {
	name         string
	hostResource string
	instanceID   string
	location     *wmi.Location
}

// VMSwitchManager manages a VM switch
type VMSwitchManager struct {
	con  *wmi.WMI
	svc  *wmi.Result
	data *wmi.Result

	exists bool
	name   string
}

func (v *vmswitchPorts) InstanceID() string {
	return strings.Replace(v.instanceID, `\`, `\\`, -1)
}

// NewVMSwitchManager returns a new VMSwitchManager type
// If the switch exists, this will return a VMSwitchManager type populated with
// the switch information
func NewVMSwitchManager(name string) (*VMSwitchManager, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}

	// Get virtual switch management service
	svc, err := w.GetOne(VMSwitchManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}

	sw := &VMSwitchManager{
		con:  w,
		svc:  svc,
		name: name,
	}
	if err := sw.refresh(); err != nil {
		return nil, err
	}
	return sw, nil
}

// Name returns the name of this VMswitch
func (s *VMSwitchManager) Name() string {
	mutex.Lock()
	defer mutex.Unlock()
	return s.name
}

func (s *VMSwitchManager) getVMSwitchManager(name string) (*wmi.Result, bool, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ElementName", Value: name, Type: wmi.Equals}},
	}
	sw, err := s.con.Gwmi(VMSwitchClass, []string{}, qParams)
	if err != nil {
		return nil, false, err
	}

	elements, err := sw.Elements()
	if err != nil {
		return nil, false, err
	}

	if len(elements) > 0 {
		return elements[0], true, nil
	}

	data, err := s.con.Get(VMSwitchSettings)
	if err != nil {
		return nil, false, err
	}
	return data, false, nil
}

func (s *VMSwitchManager) refresh() error {
	if s.name == "" {
		return fmt.Errorf("Switch name not set")
	}
	mutex.Lock()
	defer mutex.Unlock()
	var err error
	s.data, s.exists, err = s.getVMSwitchManager(s.name)
	return err
}

func (s *VMSwitchManager) setName(name string) {
	mutex.Lock()
	defer mutex.Unlock()
	s.name = name
}

// Exists returns a boolean value indicating whether or not
// the VMSwitch exists
func (s *VMSwitchManager) Exists() bool {
	s.refresh()
	return s.exists
}

// Release closes the WMI connection associated with this
// VMSWitchManager
func (s *VMSwitchManager) Release() {
	s.con.Close()
}

func (s *VMSwitchManager) getExternalPort(name string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "ElementName",
				Value: name,
				Type:  wmi.Equals},
		},
	}
	fields := []string{}
	result, err := s.con.GetOne(ExternalPort, fields, qParams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *VMSwitchManager) getDefaultSettingsData() (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "InstanceID",
				Value: "%%\\\\Default",
				Type:  wmi.Like},
		},
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: ETHConnResSubType,
				Type:  wmi.Equals},
		},
	}
	fields := []string{}
	result, err := s.con.GetOne(PortAllocSetData, fields, qParams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// SetSwitchName renames this VMSwitch
func (s *VMSwitchManager) SetSwitchName(name string) error {
	//Change switch name
	var result *wmi.Result
	var err error
	var text string

	if result, err = s.data.Get("associators_", nil, VMSwitchSettings); err != nil {
		return err
	}
	if result, err = result.ItemAtIndex(0); err != nil {
		return err
	}
	if err = result.Set("ElementName", name); err != nil {
		return err
	}
	if text, err = result.GetText(1); err != nil {
		return err
	}
	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("ModifySystemSettings", text, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call ModifySystemSettings: %v", err)
	}

	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	s.setName(name)
	return nil
}

// Delete removes this VMSwitch
func (s *VMSwitchManager) Delete() error {
	sw, err := s.data.Path()
	if err != nil {
		return fmt.Errorf("Failed to get Path: %v", err)
	}
	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("DestroySystem", sw, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DestroySystem: %v", err)
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

// Create creates this VMswitch
func (s *VMSwitchManager) Create() error {
	if s.Exists() {
		return nil
	}

	swInstance, err := s.data.Get("SpawnInstance_")
	if err != nil {
		return fmt.Errorf("Failed to call SpawnInstance_: %v", err)
	}
	err = swInstance.Set("ElementName", s.name)
	if err != nil {
		return fmt.Errorf("Failed to set switch ElementName: %v", err)
	}

	switchText, err := swInstance.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get switch text: %v", err)
	}

	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := s.svc.Get("DefineSystem", switchText, nil, nil, &resultingSystem, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	err = s.refresh()
	return err
}

func (s *VMSwitchManager) getSwitchSettings() (*wmi.Result, error) {
	if s.Exists() == false {
		return nil, fmt.Errorf("Switch %s is not yet created", s.name)
	}
	count := 0
	for {
		if count >= 50 {
			break
		}
		settingsDataResult, err := s.data.Get("associators_", nil, VMSwitchSettings)
		if err != nil {
			return nil, fmt.Errorf("Failed to get assoc: %v", err)
		}
		c, err := settingsDataResult.Elements()
		if err != nil {
			return nil, err
		}

		if len(c) > 0 {
			return c[0], nil
		}

		count++
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("Failed to get switch settings")
}

func (s *VMSwitchManager) getSwitchPorts() ([]vmswitchPorts, error) {
	settingsData, err := s.getSwitchSettings()
	if err != nil {
		return []vmswitchPorts{}, fmt.Errorf("Failed to get item2: %v", err)
	}

	ethernetPortAllocationData, err := settingsData.Get("associators_", nil, PortAllocSetData)
	if err != nil {
		return []vmswitchPorts{}, fmt.Errorf("Failed to get assoc: %v", err)
	}
	ports, err := ethernetPortAllocationData.Elements()
	if err != nil {
		return []vmswitchPorts{}, err
	}
	switchPorts := make([]vmswitchPorts, len(ports))
	for i, port := range ports {
		resource, err := port.GetProperty("HostResource")
		if err != nil {
			return []vmswitchPorts{}, err
		}
		arr := resource.ToArray()
		valueArray := arr.ToValueArray()
		if len(valueArray) == 0 {
			continue
		}

		valuePath := valueArray[0].(string)
		location, err := wmi.NewLocation(valuePath)
		if err != nil {
			return []vmswitchPorts{}, err
		}
		defer location.Close()

		extPort, err := location.GetResult()
		if err != nil {
			return []vmswitchPorts{}, err
		}

		name, err := extPort.GetProperty("ElementName")
		if err != nil {
			return []vmswitchPorts{}, err
		}

		ID, err := port.GetProperty("InstanceID")
		if err != nil {
			return []vmswitchPorts{}, err
		}
		switchPorts[i].hostResource = valuePath
		switchPorts[i].instanceID = ID.Value().(string)
		switchPorts[i].location = location
		switchPorts[i].name = name.Value().(string)

	}
	return switchPorts, nil
}

// RemoveExternalPort will remove the external port from the VMSWitch
func (s *VMSwitchManager) RemoveExternalPort() error {
	ports, err := s.getSwitchPorts()
	if err != nil {
		return err
	}
	resources := []string{}
	for _, port := range ports {
		if port.location.Class != ComputerSystem && port.location.Class != ExternalPort {
			continue
		}
		qParams := []wmi.Query{
			&wmi.AndQuery{wmi.QueryFields{Key: "InstanceID", Value: port.InstanceID(), Type: wmi.Equals}},
		}
		settings, err := s.con.GetOne(CIMResAllocSettingDataClass, []string{}, qParams)
		if err != nil {
			return fmt.Errorf("Failed to run query: %v", err)
		}
		path, err := settings.Path()
		if err != nil {
			return err
		}
		resources = append(resources, path)
	}

	if len(resources) == 0 {
		return nil
	}
	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("RemoveResourceSettings", resources, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call RemoveResourceSettings: %v", err)
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *VMSwitchManager) getExternalPortSettingsData(name string) (*wmi.Result, error) {
	extPort, err := s.getExternalPort(name)
	if err != nil {
		return nil, fmt.Errorf("Failed to get external port: %v", err)
	}
	extPortPath, err := extPort.Path()
	if err != nil {
		return nil, fmt.Errorf("Could not call path_: %v", err)
	}
	extPortAlloc, err := s.getDefaultSettingsData()
	if err != nil {
		return nil, fmt.Errorf("1> %v", err)
	}

	err = extPortAlloc.Set("HostResource", []string{extPortPath})
	if err != nil {
		return nil, fmt.Errorf("Failed to set HostResource: %v", err)
	}
	return extPortAlloc, nil
}

func (s *VMSwitchManager) hasPortAttached(name string) (bool, error) {
	ports, err := s.getSwitchPorts()
	if err != nil {
		return false, err
	}

	for _, port := range ports {
		if name == port.name {
			return true, nil
		}
	}
	return false, nil
}

// SetExternalPort sets the external port on this switch
func (s *VMSwitchManager) SetExternalPort(name string) error {
	hasPort, err := s.hasPortAttached(name)
	if err != nil {
		return err
	}
	if hasPort {
		return nil
	}

	portData, err := s.getExternalPortSettingsData(name)
	if err != nil {
		return err
	}
	extText, err := portData.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get ext_port_alloc text: %v", err)
	}
	resources := []string{
		extText,
	}

	virtualEthernetSwSetData, err := s.getSwitchSettings()
	if err != nil {
		return fmt.Errorf("Failed to get item: %v", err)
	}

	path, err := virtualEthernetSwSetData.Path()
	if err != nil {
		return fmt.Errorf("Failed to get item: %v", err)
	}

	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("AddResourceSettings", path, resources, nil, &jobPath)
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
