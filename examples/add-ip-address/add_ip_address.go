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

	// See documentation on MSFT_NetIPAddress class at: https://msdn.microsoft.com/en-us/library/hh872425(v=vs.85).aspx
	netip, err := w.Get("MSFT_NetIPAddress")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create() method documentation at: https://msdn.microsoft.com/en-us/library/hh872254(v=vs.85).aspx
	_, err = netip.Get("Create", nil, "Ethernet1", "10.10.10.11", 2, 24)
	if err != nil {
		fmt.Printf("Error running Create: %v", err)
		os.Exit(1)
	}
	fmt.Println("Success!")
	return
}
