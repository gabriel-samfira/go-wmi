package virt

import (
	"fmt"
	"sync"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	// "github.com/go-ole/go-ole/oleutil"
)

var mutex = sync.RWMutex{}

type VmSwitch struct {
	con  *wmi.WMI
	svc  *wmi.WMIResult
	data *wmi.WMIResult

	resources []*ole.IDispatch
	exists    bool
	name      string
}

func getVmSwitch(name string, w *wmi.WMI) (*wmi.WMIResult, bool, error) {
	qParams := []wmi.WMIQuery{
		&wmi.WMIAndQuery{wmi.QueryFields{Key: "ElementName", Value: name, Type: wmi.Equals}},
	}
	fields := []string{}
	sw, err := w.Gwmi(VM_SWITCH, fields, qParams)
	if err != nil {
		return nil, false, err
	}
	c, err := sw.Count()
	if err != nil {
		return nil, false, err
	}
	if c > 0 {
		item, err := sw.ItemAtIndex(0)
		if err != nil {
			return nil, false, err
		}
		return item, true, nil
	}
	data, err := w.Get(VM_SWITCH_SETTINGS)
	if err != nil {
		return nil, false, err
	}
	return data, false, nil
}

// NewVmSwitch returns a new VmSwitch type
// If the switch exists, this will return a VmSwitch type populated with
// the switch information
func NewVmSwitch(name string) (*VmSwitch, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}
	resources := []*ole.IDispatch{}

	// Get virtual switch management service
	svc, err := w.GetOne(VM_SWITCH_MNGMNT_SERVICE, []string{}, []wmi.WMIQuery{})
	if err != nil {
		return nil, err
	}

	resources = append(resources, svc.Raw())
	// Get switch settings data class
	data, exists, err := getVmSwitch(name, w)
	if err != nil {
		return nil, err
	}
	resources = append(resources, data.Raw())
	sw := &VmSwitch{
		con:       w,
		svc:       svc,
		data:      data,
		exists:    exists,
		resources: resources,
		name:      name,
	}
	return sw, nil
}

func (s *VmSwitch) Exists() bool {
	return s.exists
}

func (s *VmSwitch) Release() {
	for _, i := range s.resources {
		i.Release()
	}
	s.con.Close()
}

func (s *VmSwitch) addCleanup(r *wmi.WMIResult) {
	mutex.RLock()
	s.resources = append(s.resources, r.Raw())
	mutex.RUnlock()
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
	result, err := s.con.GetOne("Msvm_ExternalEthernetPort", fields, qParams)
	if err != nil {
		return nil, err
	}
	s.addCleanup(result)
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
	s.addCleanup(result)
	return result, nil
}

func (s *VmSwitch) SetSwitchName(name string) error {
	//Change switch name
	a, _ := s.data.Get("associators_", nil, "Msvm_VirtualEthernetSwitchSettingData")
	b, _ := a.ItemAtIndex(0)
	b.Set("ElementName", "br100")
	c, err := b.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get switch text: %v", err)
	}
	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("ModifySystemSettings", c, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}

	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *VmSwitch) Delete() error {
	sw, err := s.data.Path()
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}
	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("DestroySystem", sw, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}
	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *VmSwitch) CreateNewSwitch(externalPortName string) error {
	ext_port, err := s.getExternalPort(externalPortName)
	if err != nil {
		return fmt.Errorf("Failed to get external port %v", err)
	}

	ext_port_path, err := ext_port.Path()
	if err != nil {
		return fmt.Errorf("Could not call path_: %v", err)
	}
	ext_port_alloc, err := s.getDefaultSettingsData()
	if err != nil {
		return fmt.Errorf("Could not get defaultSettingsData %v", err)
	}

	err = ext_port_alloc.Set("HostResource", []string{ext_port_path})
	if err != nil {
		return fmt.Errorf("Failed to set HostResource %v", err)
	}

	err = ext_port_alloc.Set("ElementName", s.name)
	if err != nil {
		return fmt.Errorf("Failed to set ElementName %v", err)
	}

	ext_text, err := ext_port_alloc.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get ext_port_alloc text: %v", err)
	}

	resources := []string{
		ext_text,
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
	jobState, err := s.svc.Get("DefineSystem", switchText, resources, nil, nil, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}
	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *VmSwitch) RemovePort() error {
	settingsDataResult, err := s.data.Get("associators_", nil, "Msvm_VirtualEthernetSwitchSettingData")
	if err != nil {
		return fmt.Errorf("Failed to get assoc: %v", err)
	}
	defer settingsDataResult.Release()

	settingsData, err := settingsDataResult.ItemAtIndex(0)
	if err != nil {
		return fmt.Errorf("Failed to get item: %v", err)
	}
	defer settingsData.Release()

	ethernetPortAllocationData, err := settingsData.Get("associators_", nil, "Msvm_EthernetPortAllocationSettingData")
	if err != nil {
		return fmt.Errorf("Failed to get assoc: %v", err)
	}
	defer ethernetPortAllocationData.Release()

	count, err := ethernetPortAllocationData.Count()
	if err != nil {
		return err
	}

	if count == 0 {
		// No ports to remove
		return nil
	}

	resources := []string{}
	for i := 0; i < count; i++ {
		port, err := ethernetPortAllocationData.ItemAtIndex(i)
		if err != nil {
			return fmt.Errorf("Failed to get item: %v", err)
		}
		defer port.Release()

		portPath, err := port.Path()
		if err != nil {
			return fmt.Errorf("Failed to get item: %v", err)
		}
		resources = append(resources, portPath)
	}
	fmt.Println(resources)
	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("RemoveResourceSettings", resources, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}

	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *VmSwitch) SetExternalPort(name string) error {
	ext_port, err := s.getExternalPort(name)
	if err != nil {
		return fmt.Errorf("-1> %v", err)
	}
	defer ext_port.Release()

	ext_port_path, err := ext_port.Path()
	if err != nil {
		return fmt.Errorf("Could not call path_: %v", err)
	}
	ext_port_alloc, err := s.getDefaultSettingsData()
	if err != nil {
		return fmt.Errorf("1> %v", err)
	}
	defer ext_port_alloc.Release()

	err = ext_port_alloc.Set("HostResource", []string{ext_port_path})
	if err != nil {
		return fmt.Errorf("2> %v", err)
	}

	ext_text, err := ext_port_alloc.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get ext_port_alloc text: %v", err)
	}

	resources := []string{
		ext_text,
	}

	a, err := s.data.Get("associators_", nil, "Msvm_VirtualEthernetSwitchSettingData")
	if err != nil {
		return fmt.Errorf("Failed to get assoc: %v", err)
	}
	defer a.Release()

	b, err := a.ItemAtIndex(0)
	if err != nil {
		return fmt.Errorf("Failed to get item: %v", err)
	}
	defer b.Release()

	c, err := b.Path()
	if err != nil {
		return fmt.Errorf("Failed to get item: %v", err)
	}

	jobPath := ole.VARIANT{}
	jobState, err := s.svc.Get("AddResourceSettings", c, resources, nil, &jobPath)
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}

	if jobState.Value().(int32) == wmi.WMI_JOB_STATUS_STARTED {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return err
		}
	}
	return nil
}
