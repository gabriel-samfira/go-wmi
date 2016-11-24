package main

import (
	"encoding/json"
	"fmt"

	virt "github.com/gabriel-samfira/go-wmi/virt/network"
)

func main() {
	adapters, err := virt.GetNetworkAdapters("Ethernet0")
	if err != nil {
		fmt.Println(err)
		return
	}
	m, err := json.MarshalIndent(&adapters, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(m), err)
	if len(adapters) > 0 {
		a := adapters[0]
		v, err := a.GetIPAddresses()
		if err != nil {
			fmt.Println(err)
			return
		}
		n, err := json.MarshalIndent(&v, "", "  ")
		fmt.Println(string(n), err)
	}
}
