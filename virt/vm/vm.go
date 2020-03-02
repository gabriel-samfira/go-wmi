package vm

import (
	"fmt"
	"go-wmi/wmi"
	"runtime"
	"strconv"
	"strings"

	"go-wmi/utils"

	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

func addResourceSetting(svc *wmi.Result, settingsData []string, vmPath string) ([]string, error) {
	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := svc.Get("AddResourceSettings", vmPath, settingsData, &resultingSystem, &jobPath)
	if err != nil {
		return nil, errors.Wrap(err, "calling ModifyResourceSettings")
	}

	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return nil, errors.Wrap(err, "waiting for job")
		}
	}
	safeArrayConversion := resultingSystem.ToArray()
	valArray := safeArrayConversion.ToValueArray()
	if len(valArray) == 0 {
		return nil, fmt.Errorf("no resource in resultingSystem value")
	}
	resultingSystems := make([]string, len(valArray))
	for idx, val := range valArray {
		resultingSystems[idx] = val.(string)
	}
	return resultingSystems, nil
}

func getResourceAllocSettings(con *wmi.WMI, resourceSubType string, class string) (*wmi.Result, error) {
	if class == "" {
		class = ResourceAllocSettingDataClass
	}

	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: resourceSubType,
				Type:  wmi.Equals,
			},
		},
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "InstanceID",
				Value: "%\\\\Default",
				Type:  wmi.Like,
			},
		},
	}
	settingsDataResults, err := con.Gwmi(class, []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("getting %s", class))
	}
	settingsData, err := settingsDataResults.ItemAtIndex(0)
	if err != nil {
		return nil, errors.Wrap(err, "getting result")
	}
	return settingsData, nil
}

// NewVMManager returns a new Manager type
func NewVMManager() (*Manager, error) {
	w, err := wmi.NewConnection(".", `root\virtualization\v2`)
	if err != nil {
		return nil, err
	}

	// Get virtual machine management service
	svc, err := w.GetOne(VMManagementService, []string{}, []wmi.Query{})
	if err != nil {
		return nil, err
	}

	sw := &Manager{
		con: w,
		svc: svc,
	}
	return sw, nil
}

// Manager manages a VM switch
type Manager struct {
	con *wmi.WMI
	svc *wmi.Result
}

// GetVM returns the virtual machine identified by instanceID
func (m *Manager) GetVM(instanceID string) (*VirtualMachine, error) {
	fields := []string{}
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "VirtualSystemType",
				Value: VirtualSystemTypeRealized,
				Type:  wmi.Equals},
		},
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "VirtualSystemIdentifier",
				Value: instanceID,
				Type:  wmi.Equals},
		},
	}

	result, err := m.con.Gwmi(VirtualSystemSettingDataClass, fields, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "VirtualSystemSettingDataClass")
	}

	vssd, err := result.ItemAtIndex(0)
	if err != nil {
		return nil, errors.Wrap(err, "fetching element")
	}
	cs, err := vssd.Get("associators_", nil, ComputerSystemClass)
	if err != nil {
		return nil, errors.Wrap(err, "getting ComputerSystemClass")
	}
	elem, err := cs.Elements()
	if err != nil || len(elem) == 0 {
		return nil, errors.Wrap(err, "getting elements")
	}
	pth, err := elem[0].Path()
	if err != nil {
		return nil, errors.Wrap(err, "VM path")
	}
	return &VirtualMachine{
		mgr:                m,
		activeSettingsData: vssd,
		computerSystem:     elem[0],
		path:               pth,
	}, nil
}

// ListVM returns a list of virtual machines
func (m *Manager) ListVM() ([]*VirtualMachine, error) {
	fields := []string{}
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "VirtualSystemType",
				Value: VirtualSystemTypeRealized,
				Type:  wmi.Equals},
		},
	}

	result, err := m.con.Gwmi(VirtualSystemSettingDataClass, fields, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "VirtualSystemSettingDataClass")
	}

	elements, err := result.Elements()
	if err != nil {
		return nil, errors.Wrap(err, "Elements")
	}
	vms := make([]*VirtualMachine, len(elements))
	for idx, val := range elements {
		cs, err := val.Get("associators_", nil, ComputerSystemClass)
		if err != nil {
			return nil, errors.Wrap(err, "getting ComputerSystemClass")
		}
		elem, err := cs.Elements()
		if err != nil || len(elem) == 0 {
			return nil, errors.Wrap(err, "getting elements")
		}
		vms[idx] = &VirtualMachine{
			mgr:                m,
			activeSettingsData: val,
			computerSystem:     elem[0],
		}
	}
	return vms, nil
}

// CreateVM creates a new virtual machine
func (m *Manager) CreateVM(name string, memoryMB int64, cpus int, limitCPUFeatures bool, notes []string, generation GenerationType, secureBoot bool) (*VirtualMachine, error) {
	vmSettingsDataInstance, err := m.con.Get(VirtualSystemSettingDataClass)
	if err != nil {
		return nil, err
	}

	newVMInstance, err := vmSettingsDataInstance.Get("SpawnInstance_")
	if err != nil {
		return nil, errors.Wrap(err, "calling SpawnInstance_")
	}

	if err := newVMInstance.Set("ElementName", name); err != nil {
		return nil, errors.Wrap(err, "Set ElementName")
	}
	if err := newVMInstance.Set("VirtualSystemSubType", string(generation)); err != nil {
		return nil, errors.Wrap(err, "Set VirtualSystemSubType")
	}

	if generation == Generation2 {
		if err := newVMInstance.Set("SecureBootEnabled", secureBoot); err != nil {
			return nil, errors.Wrap(err, "Set VirtualSystemSubType")
		}
	}

	if notes != nil && len(notes) > 0 {
		// Don't ask...
		// Well, ok...if you must. The Msvm_VirtualSystemSettingData has a Notes
		// property of type []string. But in reality, it only cares about the first
		// element of that array. So we join the notes into one newline delimited
		// string, and set that as the first and only element in a new []string{}
		vmNotes := []string{strings.Join(notes, "\n")}
		if err := newVMInstance.Set("Notes", vmNotes); err != nil {
			return nil, errors.Wrap(err, "Set Notes")
		}
	}

	vmText, err := newVMInstance.GetText(1)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get VM instance XML")
	}

	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := m.svc.Get("DefineSystem", vmText, nil, nil, &resultingSystem, &jobPath)
	if err != nil {
		return nil, errors.Wrap(err, "calling DefineSystem")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return nil, errors.Wrap(err, "waiting for job")
		}
	}

	// The resultingSystem value for DefineSystem is always a string containing the
	// location of the newly created resource
	locationURI := resultingSystem.Value().(string)
	loc, err := wmi.NewLocation(locationURI)
	if err != nil {
		return nil, errors.Wrap(err, "getting location")
	}

	result, err := loc.GetResult()
	if err != nil {
		return nil, errors.Wrap(err, "getting result")
	}

	// The name field of the returning class is actually the InstanceID...
	id, err := result.GetProperty("Name")
	if err != nil {
		return nil, errors.Wrap(err, "fetching VM ID")
	}

	vm, err := m.GetVM(id.Value().(string))
	if err != nil {
		return nil, errors.Wrap(err, "fetching VM")
	}

	if err := vm.SetMemory(memoryMB); err != nil {
		return nil, errors.Wrap(err, "setting memory limit")
	}

	if err := vm.SetNumCPUs(cpus); err != nil {
		return nil, errors.Wrap(err, "setting CPU limit")
	}

	bootOrder := []int32{
		int32(BootHDD),
		int32(BootPXE),
		int32(BootCDROM),
		int32(BootFloppy),
	}

	if err := vm.SetBootOrder(bootOrder); err != nil {
		return nil, errors.Wrap(err, "setting boot order")
	}

	return vm, nil
}

// Release closes the WMI connection associated with this
// Manager
func (m *Manager) Release() {
	m.con.Close()
}

// VirtualMachine represents a single virtual machine
type VirtualMachine struct {
	mgr *Manager

	activeSettingsData *wmi.Result
	computerSystem     *wmi.Result
	path               string
}

// Name returns the current name of this virtual machine
func (v *VirtualMachine) Name() (string, error) {
	name, err := v.computerSystem.GetProperty("ElementName")
	if err != nil {
		return "", errors.Wrap(err, "getting ElementName")
	}
	return name.Value().(string), nil
}

// ID returns the instance ID of this Virtual machine
func (v *VirtualMachine) ID() (string, error) {
	id, err := v.activeSettingsData.GetProperty("VirtualSystemIdentifier")
	if err != nil {
		return "", errors.Wrap(err, "fetching VM ID")
	}
	return id.Value().(string), nil
}

// AttachDisks attaches the supplied disks, to this virtual machine
func (v *VirtualMachine) AttachDisks(disks []string) error {
	return nil
}

// SetBootOrder sets the VM boot order
func (v *VirtualMachine) SetBootOrder(bootOrder []int32) error {
	if err := v.activeSettingsData.Set("BootOrder", bootOrder); err != nil {
		return errors.Wrap(err, "Set BootOrder")
	}

	vmText, err := v.activeSettingsData.GetText(1)

	jobPath := ole.VARIANT{}
	jobState, err := v.mgr.svc.Get("ModifySystemSettings", vmText, &jobPath)
	if err != nil {
		return errors.Wrap(err, "calling ModifySystemSettings")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "waiting for job")
		}
	}
	return nil
}

func (v *VirtualMachine) modifyResourceSettings(settings []string) error {
	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := v.mgr.svc.Get("ModifyResourceSettings", settings, &resultingSystem, &jobPath)
	if err != nil {
		return errors.Wrap(err, "calling ModifyResourceSettings")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "waiting for job")
		}
	}
	return nil
}

// SetMemory sets the virtual machine memory allocation
func (v *VirtualMachine) SetMemory(memoryMB int64) error {
	memorySettingsResults, err := v.activeSettingsData.Get("associators_", nil, MemorySettingDataClass)
	if err != nil {
		return errors.Wrap(err, "getting MemorySettingDataClass")
	}

	memorySettings, err := memorySettingsResults.ItemAtIndex(0)
	if err != nil {
		return errors.Wrap(err, "ItemAtIndex")
	}

	if err := memorySettings.Set("Limit", memoryMB); err != nil {
		return errors.Wrap(err, "Limit")
	}

	if err := memorySettings.Set("Reservation", memoryMB); err != nil {
		return errors.Wrap(err, "Reservation")
	}

	if err := memorySettings.Set("VirtualQuantity", memoryMB); err != nil {
		return errors.Wrap(err, "VirtualQuantity")
	}

	memText, err := memorySettings.GetText(1)
	if err != nil {
		return errors.Wrap(err, "Failed to get VM instance XML")
	}

	return v.modifyResourceSettings([]string{memText})
}

// SetNumCPUs sets the number of CPU cores on the VM
func (v *VirtualMachine) SetNumCPUs(cpus int) error {
	hostCpus := runtime.NumCPU()
	if hostCpus < cpus {
		return fmt.Errorf("Number of cpus exceeded available host resources")
	}

	procSettingsResults, err := v.activeSettingsData.Get("associators_", nil, ProcessorSettingDataClass)
	if err != nil {
		return errors.Wrap(err, "getting ProcessorSettingDataClass")
	}

	procSettings, err := procSettingsResults.ItemAtIndex(0)
	if err != nil {
		return errors.Wrap(err, "ItemAtIndex")
	}

	if err := procSettings.Set("VirtualQuantity", uint64(cpus)); err != nil {
		return errors.Wrap(err, "VirtualQuantity")
	}

	if err := procSettings.Set("Reservation", cpus); err != nil {
		return errors.Wrap(err, "Reservation")
	}

	if err := procSettings.Set("Limit", 100000); err != nil {
		return errors.Wrap(err, "Limit")
	}

	procText, err := procSettings.GetText(1)
	if err != nil {
		return errors.Wrap(err, "Failed to get VM instance XML")
	}
	return v.modifyResourceSettings([]string{procText})
}

// SetPowerState sets the desired power state on a virtual machine.
func (v *VirtualMachine) SetPowerState(state PowerState) error {
	jobPath := ole.VARIANT{}
	jobState, err := v.computerSystem.Get("RequestStateChange", uint16(state), &jobPath)
	if err != nil {
		return errors.Wrap(err, "calling RequestStateChange")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err = wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "waiting for job")
		}
	}
	return nil
}

// CreateNewSCSIController will create a new ISCSI controller on this VM
func (v *VirtualMachine) CreateNewSCSIController() (string, error) {
	resData, err := getResourceAllocSettings(v.mgr.con, SCSIControllerResSubType, ResourceAllocSettingDataClass)
	if err != nil {
		return "", errors.Wrap(err, "getResourceAllocSettings")
	}
	newID, err := utils.UUID4()
	if err != nil {
		return "", errors.Wrap(err, "UUID4")
	}
	if err := resData.Set("VirtualSystemIdentifiers", []string{fmt.Sprintf("{%s}", newID)}); err != nil {
		return "", errors.Wrap(err, "VirtualSystemIdentifiers")
	}

	dataText, err := resData.GetText(1)
	if err != nil {
		return "", errors.Wrap(err, "GetText")
	}

	resCtrl, err := addResourceSetting(v.mgr.svc, []string{dataText}, v.path)
	if err != nil {
		return "", errors.Wrap(err, "addResourceSetting")
	}
	return resCtrl[0], nil
}

func (v *VirtualMachine) getResourceOfType(subType string) ([]string, error) {

	settingClasses, err := v.activeSettingsData.Get("associators_", nil, ResourceAllocSettingDataClass)
	if err != nil {
		return nil, errors.Wrap(err, "getting ResourceAllocSettingDataClass")
	}
	settingElements, err := settingClasses.Elements()
	if err != nil {
		return nil, errors.Wrap(err, "fetching elements")
	}

	ret := []string{}
	for _, val := range settingElements {
		resSubtype, err := val.GetProperty("ResourceSubType")
		if err != nil {
			continue
		}
		if resSubtype.Value().(string) == subType {
			pth, err := val.Path()
			if err != nil {
				return nil, errors.Wrap(err, "subType path_")
			}
			ret = append(ret, pth)
		}
	}
	return ret, nil
}

// GetSCSIControllers will return a list of SCSI controller paths
func (v *VirtualMachine) GetSCSIControllers() ([]SCSIController, error) {
	res, err := v.getResourceOfType(SCSIControllerResSubType)
	if err != nil {
		return nil, errors.Wrap(err, "getResourceOfType SCSIControllerResSubType")
	}
	vmPath, err := v.computerSystem.Path()
	if err != nil {
		return nil, errors.Wrap(err, "vmPath")
	}
	ret := make([]SCSIController, len(res))
	for idx, val := range res {
		ret[idx] = SCSIController{
			mgr:    v.mgr,
			path:   val,
			vmPath: vmPath,
		}
	}
	return ret, nil
}

// SCSIController represents a SCSI controller attached to a VM
type SCSIController struct {
	mgr      *Manager
	path     string
	vmPath   string
	resource *wmi.Result
}

// AttachDriveToAddress attaches a new drive to this SCSI controller, on the specified slot
func (s *SCSIController) AttachDriveToAddress(path string, driveType DriveType, address int) (string, error) {
	resData, err := getResourceAllocSettings(s.mgr.con, string(driveType), ResourceAllocSettingDataClass)
	if err != nil {
		return "", errors.Wrap(err, "getResourceOfType driveType")
	}
	if err := resData.Set("Parent", s.path); err != nil {
		return "", errors.Wrap(err, "set Parent")
	}
	if err := resData.Set("Address", strconv.Itoa(address)); err != nil {
		return "", errors.Wrap(err, "set Parent")
	}
	if err := resData.Set("AddressOnParent", strconv.Itoa(address)); err != nil {
		return "", errors.Wrap(err, "set Parent")
	}

	dataText, err := resData.GetText(1)
	if err != nil {
		return "", errors.Wrap(err, "GetText")
	}

	resCtrl, err := addResourceSetting(s.mgr.svc, []string{dataText}, s.vmPath)
	if err != nil {
		return "", errors.Wrap(err, "addResourceSetting")
	}
	drivePath := resCtrl[0]

	var diskType string
	switch driveType {
	case DiskDrive:
		diskType = IDEDiskResSubType
	case DVDDrive:
		diskType = IDEDVDResSubType
	default:
		return "", fmt.Errorf("invalid drive type")
	}

	storageRes, err := getResourceAllocSettings(s.mgr.con, diskType, StorageAllocSettingDataClass)
	if err != nil {
		return "", errors.Wrap(err, "getResourceAllocSettings")
	}

	if err := storageRes.Set("Parent", drivePath); err != nil {
		return "", errors.Wrap(err, "Parent")
	}

	if err := storageRes.Set("HostResource", []string{path}); err != nil {
		return "", errors.Wrap(err, "HostResource")
	}

	storageResText, err := storageRes.GetText(1)
	if err != nil {
		return "", errors.Wrap(err, "GetText")
	}
	resCtrl, err = addResourceSetting(s.mgr.svc, []string{storageResText}, s.vmPath)
	if err != nil {
		return "", errors.Wrap(err, "addResourceSetting storageRes")
	}

	return resCtrl[0], nil
}

// AttachDrive attaches a new drive to this SCSI controller. The slot is the first free slot
// available. If no slot is available, an error is returned.
func (s *SCSIController) AttachDrive(path string, driveType DriveType) (string, error) {
	slots, err := s.EmptySlots()
	if err != nil {
		return "", errors.Wrap(err, "EmptySlots")
	}
	if len(slots) == 0 {
		return "", fmt.Errorf("no empty slots available on controller")
	}
	slot := slots[0]
	return s.AttachDriveToAddress(path, driveType, slot)
}

// Path returns the WMI path of this controller
func (s *SCSIController) Path() string {
	return s.path
}

// EmptySlots returns a list of empty slot addresses that can be used to
// attach devices
func (s *SCSIController) EmptySlots() ([]int, error) {
	devices, err := s.AttachedDevices()
	if err != nil {
		return nil, errors.Wrap(err, "AttachedDevices")
	}

	ret := []int{}
	for i := 0; i < MaxSCSIControllerSlots; i++ {
		if _, ok := devices[i]; !ok {
			ret = append(ret, i)
		}
	}
	return ret, nil
}

// AttachedDevices returns a list of attached devices to the supplied controller
func (s *SCSIController) AttachedDevices() (map[int]string, error) {
	qParams := []wmi.Query{
		&wmi.OrQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: PhysDiskResSubType,
				Type:  wmi.Equals},
		},
		&wmi.OrQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: DiskResSubtype,
				Type:  wmi.Equals},
		},
		&wmi.OrQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: DVDResSubType,
				Type:  wmi.Equals},
		},
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "Parent",
				Value: strings.Replace(s.path, `\`, `\\`, -1),
				Type:  wmi.Equals},
		},
	}
	result, err := s.mgr.con.Gwmi(ResourceAllocSettingDataClass, []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "Gwmi")
	}
	ret := map[int]string{}
	resultElements, err := result.Elements()
	if err != nil {
		return nil, errors.Wrap(err, "Elements")
	}
	for _, val := range resultElements {
		addrProp, err := val.GetProperty("AddressOnParent")
		if err != nil {
			return nil, errors.Wrap(err, "AddressOnParent")
		}

		if addrProp.Value() == nil {
			continue
		}

		addr, err := strconv.Atoi(addrProp.Value().(string))
		if err != nil {
			continue
		}

		pth, err := val.Path()
		if err != nil {
			return nil, errors.Wrap(err, "Path_")
		}
		ret[addr] = pth
	}
	return ret, nil
}
