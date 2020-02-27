package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	virt "go-wmi/virt/network"
)

func main() {
	adapterNames := flag.String("nics", "", "adapter name to get info for")
	flag.Parse()

	var names []string
	if *adapterNames != "" {
		names = strings.Split(*adapterNames, ",")
	}
	adapters, err := virt.GetNetworkAdapters(names...)

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
		for _, nic := range adapters {
			v, err := nic.GetIPAddresses()
			if err != nil {
				fmt.Println(err)
				continue
			}
			// err = nic.Enable()
			// if err != nil {
			// 	fmt.Println(err)
			// 	return
			// }
			n, err := json.MarshalIndent(&v, "", "  ")
			fmt.Println(string(n), err)
		}
	}
}
