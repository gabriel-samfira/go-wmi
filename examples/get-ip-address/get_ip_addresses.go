package main

import (
	"fmt"
	"os"

	"go-wmi/wmi"
)

func main() {
	w, err := wmi.NewConnection(".", `Root\StandardCimv2`, nil, nil, nil, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// AddressFamily:
	// 	2 - IPv4
	//  23 - IPv6
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "AddressFamily", Value: 2, Type: wmi.Equals}},
	}
	// See documentation on MSFT_NetIPAddress class at: https://msdn.microsoft.com/en-us/library/hh872425(v=vs.85).aspx
	netip, err := w.Gwmi("MSFT_NetIPAddress", []string{}, qParams)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	elements, err := netip.Elements()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(elements) > 0 {
		for i := 0; i < len(elements); i++ {
			address, err := elements[i].GetProperty("IPAddress")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			l, err := elements[i].GetProperty("ValidLifetime")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			iface, err := elements[i].GetProperty("InterfaceAlias")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Found IP %v on interface %v --> %v\n", address.Value(), iface.Value(), l.Value())
		}
	}
	return
}
