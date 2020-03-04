package vm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/pkg/errors"
)

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
