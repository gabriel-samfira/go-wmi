package network

import (
	"sync"

	"github.com/gabriel-samfira/go-wmi/wmi"
)

// PowerManagementCapability represents an adapters power management
// capabilities.
type PowerManagementCapability uint16

// NetAdapterState is the plug and play state of the network adapter.
type NetAdapterState int32

// These are the values for the PnP adapter states. Details at:
// https://docs.microsoft.com/en-us/previous-versions/windows/desktop/legacy/hh968170(v%3Dvs.85)
const (
	AdapterUnknown NetAdapterState = iota
	AdapterPresent
	AdapterStarted
	AdapterDisabled
)

// Power management capabilities for net adapters. For details see:
// https://docs.microsoft.com/en-us/previous-versions/windows/desktop/legacy/hh968170(v%3Dvs.85)
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

// NetIPAddress is a representation of the MSFT_NetIPAddress cim class:
// https://docs.microsoft.com/en-us/previous-versions/windows/desktop/legacy/hh872425(v%3Dvs.85)
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

// NetAdapter is the equivalent of MSFT_NetAdapter. More info here:
// https://msdn.microsoft.com/en-us/library/hh968170%28v=vs.85%29.aspx?f=255&MSPPError=-2147217396
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
	InterfaceGUID                                    string
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
	LowerLayerInterfaceIndices                       []int32
	HigherLayerInterfaceIndices                      []int32
	AdminLocked                                      bool

	cimObject *wmi.Result `tag:"ignore"`
	lock      sync.Mutex  `tag:"ignore"`
}

// GetIPAddresses returns an array of NetIPAddress for this adapter
func (n *NetAdapter) GetIPAddresses() ([]NetIPAddress, error) {
	return GetNetIPAddresses(int(n.InterfaceIndex))
}

func (n *NetAdapter) callFunction(method string, params ...interface{}) (*NetAdapter, error) {
	if n.cimObject == nil {
		return nil, nil
	}

	var res *wmi.Result
	var err error
	res, err = n.cimObject.Get(method, params...)
	if err != nil {
		return nil, err
	}
	data := NetAdapter{}
	if err := wmi.PopulateStruct(res, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// Disable will disable this net adapter
func (n *NetAdapter) Disable() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.State != 2 {
		return nil
	}
	res, err := n.callFunction("Disable")
	if err != nil {
		return err
	}
	n = res
	return nil
}

// Enable will enable this net adapter
func (n *NetAdapter) Enable() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.State != 3 {
		return nil
	}
	res, err := n.callFunction("Enable")
	if err != nil {
		return err
	}
	n = res
	return nil
}

// Rename will set a new name to this net adapter
func (n *NetAdapter) Rename(name string) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.Name == name {
		return nil
	}
	res, err := n.callFunction("Rename", name)
	if err != nil {
		return err
	}
	n = res
	return nil
}

// GetNetworkAdapters returns a list of network adapters
func GetNetworkAdapters(name ...string) ([]NetAdapter, error) {
	con, err := wmi.NewStandardCimV2Connection()
	if err != nil {
		return nil, err
	}

	q := []wmi.Query{}
	if len(name) > 0 {
		for _, val := range name {
			q = append(q,
				&wmi.OrQuery{
					wmi.QueryFields{
						Key:   "Name",
						Value: val,
						Type:  wmi.Equals},
				})
		}
	}
	result, err := con.Gwmi(NetAdapterClass, []string{}, q)
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
		if err := wmi.PopulateStruct(adapter, s); err != nil {
			return []NetAdapter{}, err
		}
		s.cimObject = adapter
		ret[index] = *s
	}
	return ret, nil
}

// GetNetIPAddresses returns IP addresses for a particular
// network adapter.
func GetNetIPAddresses(index int) ([]NetIPAddress, error) {
	con, err := wmi.NewStandardCimV2Connection()
	if err != nil {
		return []NetIPAddress{}, err
	}
	defer con.Close()

	q := []wmi.Query{}
	if index != 0 {
		q = []wmi.Query{
			&wmi.AndQuery{
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
		if err := wmi.PopulateStruct(ip, s); err != nil {
			return []NetIPAddress{}, err
		}
		ret[index] = *s
	}
	return ret, nil
}
