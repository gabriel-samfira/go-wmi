package network

// Hyper-V networking constants
const (
	ExternalPort                = "Msvm_ExternalEthernetPort"
	ComputerSystem              = "Msvm_ComputerSystem"
	VMSwitchClass               = "Msvm_VirtualEthernetSwitch"
	VMSwitchSettings            = "Msvm_VirtualEthernetSwitchSettingData"
	VMSwitchManagementService   = "Msvm_VirtualEthernetSwitchManagementService"
	WIFIPort                    = "Msvm_WiFiPort"
	EthernetSwitchPort          = "Msvm_EthernetSwitchPort"
	PortAllocSetData            = "Msvm_EthernetPortAllocationSettingData"
	PortVLANSetData             = "Msvm_EthernetSwitchPortVlanSettingData"
	PortSecuritySetData         = "Msvm_EthernetSwitchPortSecuritySettingData"
	PortAllocACLSetData         = "Msvm_EthernetSwitchPortAclSettingData"
	PortExtACLSetData           = PortAllocACLSetData
	LANEndpoint                 = "Msvm_LANEndpoint"
	CIMResAllocSettingDataClass = "CIM_ResourceAllocationSettingData"
	StateDisabled               = 3
	OperationModeAccess         = 1
	OperationModeTrunk          = 2
	ETHConnResSubType           = "Microsoft:Hyper-V:Ethernet Connection"
	NetAdapterClass             = "MSFT_NetAdapter"
)
