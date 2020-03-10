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

	vs, err := vmsw.CreateVMSwitch(*switchName)
	errExit(err)

	name, err := vs.Name()
	errExit(err)
	fmt.Println(name)

	if *adapterGUID != "" {
		err = vs.SetExternalPort(*adapterGUID)
		errExit(err)
	}

	if *mgmtOS == true {
		err = vs.SetInternalPort()
		errExit(err)
	}

	// removed, err := vs.ClearExternalPort()
	// errExit(err)
	// fmt.Println(removed)

	// removed, err = vs.ClearInternalPort()
	// errExit(err)
	// fmt.Println(removed)

	// id, err := vs.ID()
	// errExit(err)
	// err = vmsw.RemoveVMSwitch(id)
	// errExit(err)
	return
}
