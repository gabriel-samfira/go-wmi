package virt

const (
	EXTERNAL_PORT                    = "Msvm_ExternalEthernetPort"
	VM_SWITCH                        = "Msvm_VirtualEthernetSwitch"
	VM_SWITCH_SETTINGS               = "Msvm_VirtualEthernetSwitchSettingData"
	VM_SWITCH_MNGMNT_SERVICE         = "Msvm_VirtualEthernetSwitchManagementService"
	WIFI_PORT                        = "Msvm_WiFiPort"
	ETHERNET_SWITCH_PORT             = "Msvm_EthernetSwitchPort"
	PORT_ALLOC_SET_DATA              = "Msvm_EthernetPortAllocationSettingData"
	PORT_VLAN_SET_DATA               = "Msvm_EthernetSwitchPortVlanSettingData"
	PORT_SECURITY_SET_DATA           = "Msvm_EthernetSwitchPortSecuritySettingData"
	PORT_ALLOC_ACL_SET_DATA          = "Msvm_EthernetSwitchPortAclSettingData"
	PORT_EXT_ACL_SET_DATA            = PORT_ALLOC_ACL_SET_DATA
	LAN_ENDPOINT                     = "Msvm_LANEndpoint"
	CIM_RES_ALLOC_SETTING_DATA_CLASS = "CIM_ResourceAllocationSettingData"
	STATE_DISABLED                   = 3
	OPERATION_MODE_ACCESS            = 1
	OPERATION_MODE_TRUNK             = 2
	ETH_CONN_RES_SUB_TYPE            = "Microsoft:Hyper-V:Ethernet Connection"
)
