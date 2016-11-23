package virt

import (
	"fmt"
	"reflect"

	"github.com/gabriel-samfira/go-wmi/wmi"
)

type PowerManagementCapability uint16

const (
	Unknown PowerManagementCapability = iota
	NotSupported
	Disabled
	Enabled
	AutoPowerSavings
	PowerStateSettable
	PowerCycling
	TimedPowerOn
)

type NetAdapter struct {
	Caption                                          string
	Description                                      string
	Name                                             string
	Status                                           string
	Availability                                     uint16
	CreationClassName                                string
	DeviceID                                         string
	ErrorCleared                                     bool
	ErrorDescription                                 string
	LastErrorCode                                    uint32
	PNPDeviceID                                      string
	PowerManagementCapabilities                      []PowerManagementCapability
	PowerManagementSupported                         bool
	StatusInfo                                       uint16
	SystemCreationClassName                          string
	SystemName                                       string
	Speed                                            string
	MaxSpeed                                         uint64
	RequestedSpeed                                   uint64
	UsageRestriction                                 uint16
	PortType                                         uint16
	OtherPortType                                    string
	OtherNetworkPortType                             string
	PortNumber                                       int32
	LinkTechnology                                   uint16
	OtherLinkTechnology                              string
	PermanentAddress                                 string
	NetworkAddresses                                 []string
	FullDuplex                                       bool
	AutoSense                                        bool
	SupportedMaximumTransmissionUnit                 uint64
	ActiveMaximumTransmissionUnit                    string
	InterfaceDescription                             string
	InterfaceName                                    string
	NetLuid                                          string
	InterfaceGuid                                    string
	InterfaceIndex                                   int32
	DeviceName                                       string
	NetLuidIndex                                     int32
	Virtual                                          bool
	Hidden                                           bool
	NotUserRemovable                                 bool
	IMFilter                                         bool
	InterfaceType                                    int32
	HardwareInterface                                bool
	WdmInterface                                     bool
	EndPointInterface                                bool
	iSCSIInterface                                   bool
	State                                            int32
	NdisMedium                                       int32
	NdisPhysicalMedium                               int32
	InterfaceOperationalStatus                       int32
	OperationalStatusDownDefaultPortNotAuthenticated bool
	OperationalStatusDownMediaDisconnected           bool
	OperationalStatusDownInterfacePaused             bool
	OperationalStatusDownLowPowerState               bool
	InterfaceAdminStatus                             int32
	MediaConnectState                                int32
	MtuSize                                          int32
	VlanID                                           int16
	TransmitLinkSpeed                                string
	ReceiveLinkSpeed                                 string
	PromiscuousMode                                  bool
	DeviceWakeUpEnable                               bool
	ConnectorPresent                                 bool
	MediaDuplexState                                 int32
	DriverDate                                       string
	DriverDateData                                   string
	DriverVersionString                              string
	DriverName                                       string
	DriverDescription                                string
	MajorDriverVersion                               int32
	MinorDriverVersion                               int32
	DriverMajorNdisVersion                           uint8
	DriverMinorNdisVersion                           uint8
	PnPDeviceID                                      string
	DriverProvider                                   string
	ComponentID                                      string
	LowerLayerInterfaceIndices                       []uint32
	HigherLayerInterfaceIndices                      []int32
	AdminLocked                                      bool
}

func populateStruct(j *wmi.WMIResult, s interface{}) error {
	valuePtr := reflect.ValueOf(s)
	elem := valuePtr.Elem()
	typeOfElem := elem.Type()

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		name := typeOfElem.Field(i).Name

		res, err := j.GetProperty(name)
		if err != nil {
			return fmt.Errorf("Failed to get property %s: %s", name, err)
		}

		wmiFieldValue := res.Value()
		if wmiFieldValue == nil {
			continue
		}

		var fieldValue interface{}
		switch name {
		case "PowerManagementCapabilities":
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]PowerManagementCapability, len(val))
				for k, v := range val {
					asString[k] = v.(PowerManagementCapability)
				}
				fieldValue = asString
			}
		case "NetworkAddresses":
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]string, len(val))
				for k, v := range val {
					asString[k] = v.(string)
				}
				fieldValue = asString
			}
		case "LowerLayerInterfaceIndices":
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]uint32, len(val))
				for k, v := range val {
					asString[k] = v.(uint32)
				}
				fieldValue = asString
			}
		case "HigherLayerInterfaceIndices":
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]int32, len(val))
				for k, v := range val {
					asString[k] = v.(int32)
				}
				fieldValue = asString
			}
		default:
			fieldValue = wmiFieldValue
		}

		v := reflect.ValueOf(fieldValue)
		if v.Kind() != field.Kind() {
			return fmt.Errorf("Invalid type returned by query for field %s: %v", name, v.Kind())
		}
		if field.CanSet() {
			field.Set(v)
		}
	}
	return nil
}

func GetNetworkAdapters(name string) ([]NetAdapter, error) {
	con, err := NewStandardCimV2Connection()

	q := []wmi.WMIQuery{}
	if name != "" {
		q = []wmi.WMIQuery{
			&wmi.WMIAndQuery{
				wmi.QueryFields{
					Key:   "Name",
					Value: name,
					Type:  wmi.Equals},
			},
		}
	}
	result, err := con.Gwmi(NET_ADAPTER_CLASS, []string{}, q)
	if err != nil {
		return []NetAdapter{}, err
	}
	adapters, err := result.Elements()
	if len(adapters) == 0 {
		return []NetAdapter{}, nil
	}
	ret := make([]NetAdapter, len(adapters))
	for index, adapter := range adapters {
		s := &NetAdapter{}
		if err := populateStruct(adapter, s); err != nil {
			return []NetAdapter{}, err
		}
		ret[index] = *s
	}
	return ret, nil
}

func NewStandardCimV2Connection() (w *wmi.WMI, err error) {
	w, err = wmi.NewConnection(".", `Root\StandardCimv2`)
	return
}
