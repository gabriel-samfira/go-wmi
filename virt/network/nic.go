package network

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

type NetIPAddress struct {
	InstanceID              string
	Caption                 string
	ElementName             string
	InstallDate             string
	StatusDescriptions      []string
	Status                  string
	HealthState             uint16
	CommunicationStatus     uint16
	DetailedStatus          uint16
	OperatingStatus         uint16
	PrimaryStatus           uint16
	OtherEnabledState       string
	RequestedState          int32
	EnabledDefault          int32
	SystemCreationClassName string
	SystemName              string
	CreationClassName       string
	Description             string
	Name                    string
	OperationalStatus       []uint16
	EnabledState            uint16
	TimeOfLastStateChange   string
	NameFormat              string
	ProtocolType            uint16
	OtherTypeDescription    string
	ProtocolIFType          int32
	IPv4Address             string
	IPv6Address             string
	Address                 string
	SubnetMask              string
	PrefixLength            uint8
	AddressType             uint16
	IPVersionSupport        uint16
	AddressOrigin           int32
	InterfaceIndex          int32
	InterfaceAlias          string
	IPAddress               string
	AddressFamily           int32
	Type                    uint8
	Store                   uint8
	PrefixOrigin            int32
	SuffixOrigin            int32
	AddressState            int32
	ValidLifetime           string
	PreferredLifetime       string
	SkipAsSource            bool
}

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
	ISCSIInterface                                   bool
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

func (n *NetAdapter) GetIPAddresses() ([]NetIPAddress, error) {
	return GetNetIPAddresses(int(n.InterfaceIndex))
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
		switch field.Interface().(type) {
		case []uint16:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]uint16, len(val))
				for k, v := range val {
					asString[k] = v.(uint16)
				}
				fieldValue = asString
			}
		case []string:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]string, len(val))
				for k, v := range val {
					asString[k] = v.(string)
				}
				fieldValue = asString
			}
		case []uint32:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]uint32, len(val))
				for k, v := range val {
					asString[k] = v.(uint32)
				}
				fieldValue = asString
			}
		case []int32:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]int32, len(val))
				for k, v := range val {
					asString[k] = v.(int32)
				}
				fieldValue = asString
			}
		case []int64:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]int64, len(val))
				for k, v := range val {
					asString[k] = v.(int64)
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

func GetNetIPAddresses(index int) ([]NetIPAddress, error) {
	con, err := NewStandardCimV2Connection()
	if err != nil {
		return []NetIPAddress{}, err
	}
	defer con.Close()

	q := []wmi.WMIQuery{}
	if index != 0 {
		q = []wmi.WMIQuery{
			&wmi.WMIAndQuery{
				wmi.QueryFields{
					Key:   "InterfaceIndex",
					Value: index,
					Type:  wmi.Equals},
			},
		}
	}
	result, err := con.Gwmi("MSFT_NetIPAddress", []string{}, q)
	if err != nil {
		return []NetIPAddress{}, err
	}
	ips, err := result.Elements()
	if len(ips) == 0 {
		return []NetIPAddress{}, nil
	}
	ret := make([]NetIPAddress, len(ips))
	for index, ip := range ips {
		s := &NetIPAddress{}
		if err := populateStruct(ip, s); err != nil {
			return []NetIPAddress{}, err
		}
		ret[index] = *s
	}
	return ret, nil
}

func NewStandardCimV2Connection() (w *wmi.WMI, err error) {
	w, err = wmi.NewConnection(".", `Root\StandardCimv2`)
	return
}
