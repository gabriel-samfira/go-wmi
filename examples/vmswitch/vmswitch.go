package main

import (
	"flag"
	"fmt"
	"os"

	virt "github.com/gabriel-samfira/go-wmi/virt/network"
)

func errExit(err error) {
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	return
}

func main() {
	adapterGUID := flag.String("nic-id", "", "nic to attach to VMswitch")
	mgmtOS := flag.Bool("mgmtOS", false, "Allow OS management")
	switchName := flag.String("vmswitch", "br100", "VM switch name")
	flag.Parse()

	vmsw, err := virt.NewVMSwitchManager()
	errExit(err)

	defer vmsw.Release()

	fmt.Printf("Creating VM switch %s\n", *switchName)
	vs, err := vmsw.CreateVMSwitch(*switchName)
	errExit(err)

	id, err := vs.ID()
	errExit(err)
	fmt.Printf("Switch ID is %s\n", id)

	if *adapterGUID != "" {
		fmt.Println("Assigning external port")
		err = vs.SetExternalPort(*adapterGUID)
		errExit(err)
	}

	if *mgmtOS == true {
		fmt.Println("Assigning internal port")
		err = vs.SetInternalPort()
		errExit(err)
	}
	fmt.Println("Removing external port")
	removed, err := vs.ClearExternalPort()
	errExit(err)
	fmt.Printf("Removed external port: %v\n", removed)

	fmt.Println("Removing internal port")
	removed, err = vs.ClearInternalPort()
	errExit(err)
	fmt.Printf("Removed internal port: %v\n", removed)

	fmt.Printf("Removing vmswitch with ID %s\n", id)
	err = vmsw.RemoveVMSwitch(id)
	errExit(err)
	return
}
