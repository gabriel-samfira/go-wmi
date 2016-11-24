package network

import (
	"fmt"
	"strings"
	"sync"
	"time"
	// "unsafe"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	// "github.com/go-ole/go-ole/oleutil"
)

var mutex = sync.RWMutex{}

type vmswitchPorts struct {
	name         string
	hostResource string
	instanceID   string
	location     *wmi.WMILocation
}

type VmSwitch struct {
	con  *wmi.WMI
	svc  *wmi.WMIResult
	data *wmi.WMIResult

	exists bool
	name   string
}

func (v *vmswitchPorts) InstanceID() string {
	return strings.Replace(v.instanceID, `\`, `\\`, -1)
}

// NewVmSwitch returns a new VmSwitch type
// If the switch exists, this will return a VmSwitch type populated with
// the switch information
func NewVmSwitch(name string) (*VmSwitch, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}

	// Get virtual switch management service
	svc, err := w.GetOne(VM_SWITCH_MNGMNT_SERVICE, []string{}, []wmi.WMIQuery{})
	if err != nil {
		return nil, err
	}

	sw := &VmSwitch{
		con:  w,
		svc:  svc,
		name: name,
	}
	if err := sw.refresh(); err != nil {
		return nil, err
	}
	return sw, nil
}

func (s *VmSwitch) Name() string {
	mutex.Lock()
	defer mutex.Unlock()
	return s.name
}

func (s *VmSwitch) getVmSwitch(name string) (*wmi.WMIResult, bool, error) {
	qParams := []wmi.WMIQuery{
		&wmi.WMIAndQuery{wmi.QueryFields{Key: "ElementName", Value: name, Type: wmi.Equals}},
	}
	sw, err := s.con.Gwmi(VM_SWITCH, []string{}, qParams)
	if err != nil {
		return nil, false, err
	}

	if elements, err := sw.Elements(); err != nil {
		return nil, false, err
	} else {
		if len(elements) > 0 {
			return elements[0], true, nil
		}
	}
	data, err := s.con.Get(VM_SWITCH_SETTINGS)
	if err != nil {
		return nil, false, err
	}
	return data, false, nil
}

func (s *VmSwitch) refresh() error {
	if s.name == "" {
		return fmt.Errorf("Switch name not set")
	}
	mutex.Lock()
	defer mutex.Unlock()
	var err error
	s.data, s.exists, err = s.getVmSwitch(s.name)
	return err
}

func (s *VmSwitch) setName(name string) {
	mutex.Lock()
	defer mutex.Unlock()
	s.name = name
}

func (s *VmSwitch) Exists() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return s.exists
}

func (s *VmSwitch) Release() {
	s.con.Close()
}

func (s *VmSwitch) getExternalPort(name string) (*wmi.WMIResult, error) {
	qParams := []wmi.WMIQuery{
		&wmi.WMIAndQuery{
			wmi.QueryFields{
				Key:   "ElementName",
				Value: name,
				Type:  wmi.Equals},
		},
	}
	fields := []string{}
	result, err := s.con.GetOne(EXTERNAL_PORT, fields, qParams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *VmSwitch) getDefaultSettingsData() (*wmi.WMIResult, error) {
	qParams := []wmi.WMIQuery{
		&wmi.WMIAndQuery{
			wmi.QueryFields{
				Key:   "InstanceID",
				Value: "%%\\\\Default",
				Type:  wmi.Like},
		},
		&wmi.WMIAndQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: ETH_CONN_RES_SUB_TYPE,
				Type:  wmi.Equals},
		},
	}
	fields := []string{}
	result, err := s.con.GetOne(PORT_ALLOC_SET_DATA, fields, qParams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *VmSwitch) SetSwitchName(name string) error {
	//Change switch name
	var result *wmi.WMIResult
	var err error
	var text string

	if result, err = s.data.Get("associators_", nil, VM_SWITCH_SETTINGS); err != nil {
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

	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	s.setName(name)
	return nil
}

func (s *VmSwitch) Delete() error {
	sw, err := s.data.Path()
	if err != nil {
		return fmt.Errorf("Failed to get Path: %v", err)
	}
	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("DestroySystem", sw, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DestroySystem: %v", err)
	}
	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *VmSwitch) Create() error {
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
	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	err = s.refresh()
	return err
}

func (s *VmSwitch) getSwitchSettings() (*wmi.WMIResult, error) {
	if s.Exists() == false {
		return nil, fmt.Errorf("Switch %s is not yet created", s.name)
	}
	count := 0
	for {
		if count >= 50 {
			break
		}
		settingsDataResult, err := s.data.Get("associators_", nil, VM_SWITCH_SETTINGS)
		if err != nil {
			return nil, fmt.Errorf("Failed to get assoc: %v", err)
		}
		if c, err := settingsDataResult.Elements(); err != nil {
			return nil, err
		} else {
			if len(c) > 0 {
				return c[0], nil
			}
		}
		count += 1
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("Failed to get switch settings")
}

func (s *VmSwitch) getSwitchPorts() ([]vmswitchPorts, error) {
	settingsData, err := s.getSwitchSettings()
	if err != nil {
		return []vmswitchPorts{}, fmt.Errorf("Failed to get item2: %v", err)
	}

	ethernetPortAllocationData, err := settingsData.Get("associators_", nil, PORT_ALLOC_SET_DATA)
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
		location, err := wmi.NewWMILocation(valuePath)
		if err != nil {
			return []vmswitchPorts{}, err
		}
		defer location.Close()

		ext_port, err := location.GetWMIResult()
		if err != nil {
			return []vmswitchPorts{}, err
		}

		name, err := ext_port.GetProperty("ElementName")
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

func (s *VmSwitch) RemovePort() error {
	ports, err := s.getSwitchPorts()
	if err != nil {
		return err
	}
	resources := []string{}
	for _, port := range ports {
		if port.location.Class != COMPUTER_SYSTEM && port.location.Class != EXTERNAL_PORT {
			continue
		}
		qParams := []wmi.WMIQuery{
			&wmi.WMIAndQuery{wmi.QueryFields{Key: "InstanceID", Value: port.InstanceID(), Type: wmi.Equals}},
		}
		settings, err := s.con.GetOne(CIM_RES_ALLOC_SETTING_DATA_CLASS, []string{}, qParams)
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
	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *VmSwitch) getExternalPortSettingsData(name string) (*wmi.WMIResult, error) {
	ext_port, err := s.getExternalPort(name)
	if err != nil {
		return nil, fmt.Errorf("Failed to get external port: %v", err)
	}
	ext_port_path, err := ext_port.Path()
	if err != nil {
		return nil, fmt.Errorf("Could not call path_: %v", err)
	}
	ext_port_alloc, err := s.getDefaultSettingsData()
	if err != nil {
		return nil, fmt.Errorf("1> %v", err)
	}

	err = ext_port_alloc.Set("HostResource", []string{ext_port_path})
	if err != nil {
		return nil, fmt.Errorf("Failed to set HostResource: %v", err)
	}
	return ext_port_alloc, nil
}

func (s *VmSwitch) hasPortAttached(name string) (bool, error) {
	if ports, err := s.getSwitchPorts(); err == nil {
		for _, port := range ports {
			if name == port.name {
				return true, nil
			}
		}
		return false, nil
	} else {
		return false, err
	}
}

func (s *VmSwitch) SetExternalPort(name string) error {
	if hasPort, err := s.hasPortAttached(name); err != nil {
		return err
	} else {
		if hasPort {
			return nil
		}
	}
	port_data, err := s.getExternalPortSettingsData(name)
	if err != nil {
		return err
	}
	ext_text, err := port_data.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get ext_port_alloc text: %v", err)
	}
	resources := []string{
		ext_text,
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

	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}
