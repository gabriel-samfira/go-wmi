package virt

import (
	"fmt"
	"strings"
	"sync"
	// "unsafe"

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

	// Get virtual switch management service
	svc, err := w.GetOne(VM_SWITCH_MNGMNT_SERVICE, []string{}, []wmi.WMIQuery{})
	if err != nil {
		return nil, err
	}

	// Get switch settings data class
	data, exists, err := getVmSwitch(name, w)
	if err != nil {
		return nil, err
	}
	sw := &VmSwitch{
		con:    w,
		svc:    svc,
		data:   data,
		exists: exists,
		name:   name,
	}
	return sw, nil
}

func (s *VmSwitch) Exists() bool {
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
	result, err := s.con.GetOne("Msvm_ExternalEthernetPort", fields, qParams)
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

	settingsData, err := settingsDataResult.ItemAtIndex(0)
	if err != nil {
		return fmt.Errorf("Failed to get item2: %v", err)
	}

	ethernetPortAllocationData, err := settingsData.Get("associators_", nil, "Msvm_EthernetPortAllocationSettingData")
	if err != nil {
		return fmt.Errorf("Failed to get assoc: %v", err)
	}

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
		resource, err := port.GetProperty("HostResource")
		if err != nil {
			return err
		}
		arr := resource.ToArray()
		valueArray := arr.ToValueArray()
		if len(valueArray) == 0 {
			continue
		}
		valuePath := valueArray[0].(string)
		parsed, err := wmi.NewPathParser(valuePath)
		if err != nil {
			return err
		}
		if strings.ToLower(parsed.Class) != "msvm_computersystem" && strings.ToLower(parsed.Class) != "msvm_externalethernetport" {
			continue
		}
		ID, err := port.GetProperty("InstanceID")
		if err != nil {
			return err
		}
		qParams := []wmi.WMIQuery{
			&wmi.WMIAndQuery{wmi.QueryFields{Key: "InstanceID", Value: strings.Replace(ID.Value().(string), `\`, `\\`, -1), Type: wmi.Equals}},
		}
		fields := []string{}
		settings, err := s.con.GetOne(CIM_RES_ALLOC_SETTING_DATA_CLASS, fields, qParams)
		if err != nil {
			return fmt.Errorf("Failed to run query: %v", err)
		}
		path, err := settings.Path()
		if err != nil {
			return err
		}
		resources = append(resources, path)
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

func (s *VmSwitch) SetExternalPort(name string) error {
	ext_port, err := s.getExternalPort(name)
	if err != nil {
		return fmt.Errorf("-1> %v", err)
	}
	ext_port_path, err := ext_port.Path()
	if err != nil {
		return fmt.Errorf("Could not call path_: %v", err)
	}
	ext_port_alloc, err := s.getDefaultSettingsData()
	if err != nil {
		return fmt.Errorf("1> %v", err)
	}

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

	virtualEthernetSwSetDataAssoc, err := s.data.Get("associators_", nil, "Msvm_VirtualEthernetSwitchSettingData")
	if err != nil {
		return fmt.Errorf("Failed to get assoc: %v", err)
	}

	virtualEthernetSwSetData, err := virtualEthernetSwSetDataAssoc.ItemAtIndex(0)
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
