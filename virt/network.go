package virt

import (
	"fmt"
	"time"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	// "github.com/go-ole/go-ole/oleutil"
)

type VmSitch struct {
	con  *wmi.WMI
	svc  *wmi.WMIResult
	data *wmi.WMIResult

	resources []*ole.IDispatch
	exists    bool
	name      string
}

func getVmSwitch(name string, w *wmi.WMI) (*wmi.WMIResult, bool, error) {
	qParams := []wmi.WMIQuery{
		wmi.WMIAndQuery{Key: "ElementName", Value: name, Type: wmi.Equals},
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

// NewVmSwitch returns a VmSwitch type populated with the necesary components
// to create a new VMSwitch
func NewVmSwitch(name string) (*VmSitch, error) {
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
	sw := &VmSitch{
		con:       w,
		svc:       svc,
		data:      data,
		exists:    exists,
		resources: resources,
		name:      name,
	}
	return sw, nil
}

func (s *VmSitch) Exists() bool {
	return s.exists
}

func (s *VmSitch) Release() {
	for _, i := range s.resources {
		i.Release()
	}
}

func (s *VmSitch) getExternalPort(name string) (*wmi.WMIResult, error) {
	qParams := []wmi.WMIQuery{
		wmi.WMIAndQuery{Key: "ElementName", Value: name, Type: wmi.Equals},
	}
	fields := []string{}
	result, err := s.con.GetOne("Msvm_ExternalEthernetPort", fields, qParams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *VmSitch) getHostObj() (*wmi.WMIResult, error) {
	qParams := []wmi.WMIQuery{
		wmi.WMIAndQuery{Key: "Description", Value: "Microsoft Hosting Computer System", Type: wmi.Equals},
	}
	fields := []string{}
	result, err := s.con.GetOne("Msvm_ComputerSystem", fields, qParams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *VmSitch) getDefaultSettingsData(class, res_sub_type string) (*wmi.WMIResult, error) {
	qParams := []wmi.WMIQuery{
		wmi.WMIAndQuery{Key: "InstanceID", Value: "%%\\\\Default", Type: wmi.Like},
		wmi.WMIAndQuery{Key: "ResourceSubType", Value: res_sub_type, Type: wmi.Equals},
	}
	fields := []string{}
	res, err := s.con.GetOne(class, fields, qParams)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *VmSitch) getSettingsData(class, instance, res_sub_type string) (*wmi.WMIResult, error) {
	qParams := []wmi.WMIQuery{
		wmi.WMIAndQuery{Key: "InstanceID", Value: instance, Type: wmi.Equals},
		wmi.WMIAndQuery{Key: "ResourceSubType", Value: res_sub_type, Type: wmi.Equals},
	}
	fields := []string{}
	res, err := s.con.GetOne(class, fields, qParams)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *VmSitch) SetExternalPort(name string) error {
	// fmt.Println(s.Exists())
	ext_port, err := s.getExternalPort(name)
	if err != nil {
		return fmt.Errorf("-1> %v", err)
	}
	s.resources = append(s.resources, ext_port.Raw())
	w, err := ext_port.GetProperty("PermanentAddress")

	if err != nil {
		return fmt.Errorf("-2> %v", err)
	}
	ext_port_path, err := ext_port.Path()
	// fmt.Println(ext_port_path)
	if err != nil {
		return fmt.Errorf("Could not call path_: %v", err)
	}
	ext_port_alloc, err := s.getDefaultSettingsData(PORT_ALLOC_SET_DATA, ETH_CONN_RES_SUB_TYPE)
	if err != nil {
		return fmt.Errorf("1> %v", err)
	}
	s.resources = append(s.resources, ext_port_alloc.Raw())

	//////////
	err = ext_port_alloc.Set("HostResource", []string{ext_port_path})
	if err != nil {
		return fmt.Errorf("2> %v", err)
	}
	/////////

	err = ext_port_alloc.Set("ElementName", s.name)
	if err != nil {
		return fmt.Errorf("3> %v", err)
	}

	int_port, err := s.getDefaultSettingsData(PORT_ALLOC_SET_DATA, ETH_CONN_RES_SUB_TYPE)
	if err != nil {
		return fmt.Errorf("4> %v", err)
	}
	s.resources = append(s.resources, int_port.Raw())

	host, err := s.getHostObj()
	if err != nil {
		return fmt.Errorf("Failed to get host object: %v", err)
	}
	s.resources = append(s.resources, host.Raw())

	host_path, err := host.Path()
	if err != nil {
		return fmt.Errorf("Failed to get host path: %v", err)
	}
	// fmt.Println(host_path)
	err = int_port.Set("HostResource", []string{host_path})
	if err != nil {
		return fmt.Errorf("Failed to set host HostResource: %v", err)
	}
	err = int_port.Set("ElementName", s.name)
	if err != nil {
		return fmt.Errorf("Failed to set host ElementName: %v", err)
	}

	err = int_port.Set("Address", w.Value())
	if err != nil {
		return fmt.Errorf("Failed to set host Address: %v", err)
	}
	ext_text, err := ext_port_alloc.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get ext_port_alloc text: %v", err)
	}

	int_port_text, err := int_port.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get int_port text: %v", err)
	}

	resources := []string{
		ext_text,
	}
	swInstance, err := s.data.Get("SpawnInstance_")
	if err != nil {
		return fmt.Errorf("Failed to Create: %v", err)
	}
	err = swInstance.Set("ElementName", s.name)
	if err != nil {
		return fmt.Errorf("Failed to set host ElementName: %v", err)
	}

	switchText, err := swInstance.GetText(1)
	if err != nil {
		return fmt.Errorf("Failed to get switch text: %v %v %v %v", err, switchText, ext_text, int_port_text)
	}

	rawSys, err := s.svc.Get("DefineSystem", switchText, resources)
	if err != nil {
		return fmt.Errorf("Failed to call DefineSystem: %v", err)
	}
	time.Sleep(5 * time.Second)
	return nil
}
