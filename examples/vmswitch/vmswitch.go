package main

import (
	"flag"
	"fmt"
	"os"

	virt "github.com/gabriel-samfira/go-wmi/virt/network"
	// wmi "github.com/gabriel-samfira/go-wmi/wmi"
)

func main() {
	adapterName := flag.String("nic", "", "nic to attach to VMswitch")
	switchName := flag.String("vmswitch", "br100", "VM switch name")
	flag.Parse()

	if *adapterName == "" {
		fmt.Println("missing net adapter name")
		os.Exit(1)
	}

	vmsw, err := virt.NewVMSwitchManager(*switchName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	name := *adapterName

	fmt.Printf("Creating %s\r\n", *switchName)
	if err := vmsw.Create(); err != nil {
		fmt.Println(err)
		return
	}

	// newName := "newName"

	// fmt.Printf("Setting VMswitch name to: %s\r\n", newName)
	// if err := vmsw.SetSwitchName(newName); err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	fmt.Printf("Setting external port to: %s\r\n", name)
	if err := vmsw.SetExternalPort(name); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Removing ports from %s\r\n", vmsw.Name())
	if err := vmsw.RemoveExternalPort(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("deleting %s\r\n", vmsw.Name())
	if err := vmsw.Delete(); err != nil {
		fmt.Println(err)
		return
	}
	vmsw.Release()
}
