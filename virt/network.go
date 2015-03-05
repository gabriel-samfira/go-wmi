package virt

import (
	"fmt"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/mattn/go-ole"
	"github.com/mattn/go-ole/oleutil"
)

type VmSitch struct {
	con  *wmi.WMI
	svc  *ole.IDispatch
	data *ole.IDispatch

	resources []*ole.IDispatch
	exists    bool
}

func getVmSwitch(name string, w *wmi.WMI) (*ole.IDispatch, bool, error) {
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
		return item.Raw(), true, nil
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
	w, err := wmi.NewConnection(".", "root\\virtualization\\v2")
	if err != nil {
		return nil, err
	}

	resources := []*ole.IDispatch{}

	// Get virtual switch management service
	svc, err := w.Get(VM_SWITCH_MNGMNT_SERVICE)
	if err != nil {
		return nil, err
	}
	resources = append(resources, svc)
	// Get switch settings data class
	data, exists, err := getVmSwitch(name, w)
	if err != nil {
		return nil, err
	}
	resources = append(resources, data)
	sw := &VmSitch{
		con:       w,
		svc:       svc,
		data:      data,
		exists:    exists,
		resources: resources,
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

func (s *VmSitch) SetExternalPort(name string) error {
	ext_port, err := s.getExternalPort(name)
	if err != nil {
		return err
	}
	// rawExtPort := ext_port.Raw()

	ext_port_path, err := ext_port.Get("GetObjectText_")
	if err != nil {
		return fmt.Errorf("Could not call path_: %v", err)
	}
	v := ext_port_path.Value()
	fmt.Printf("---------> %v\r\n", v)
	ext_port_path_val, err := oleutil.CallMethod(ext_port_path.Raw(), "path_")
	if err != nil {
		return fmt.Errorf("Could not call __PATH: %v", err)
	}
	xx := ext_port_path_val.Value()
	fmt.Printf("Value Of: %v\r\n", xx)

	port_alloc, err := s.getDefaultSettingsData(PORT_ALLOC_SET_DATA, ETH_CONN_RES_SUB_TYPE)
	if err != nil {
		return err
	}
	type cucu interface{}
	// ext_pot_path_val := ext_port_path.Value()
	// x := []string{
	// 	"asdsa",
	// }
	fmt.Printf(">>> %T\r\n", ext_port_path.Value())
	cur, err := port_alloc.GetProperty("HostResource")
	if err != nil {
		return fmt.Errorf("Could not get HostResource: %v", err)
	}
	fmt.Printf("SDASDAS: %v\r\n", cur.Value())
	err = port_alloc.Set("HostResource", ext_port_path)
	if err != nil {
		return err
	}
	return nil
}
