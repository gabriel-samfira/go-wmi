package main

import (
	"fmt"
	"os"

	vm "github.com/gabriel-samfira/go-wmi/virt/vm"
)

func errExit(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func main() {
	vmm, err := vm.NewVMManager()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer vmm.Release()

	vms, err := vmm.ListVM()
	if err != nil {
		errExit(err)
	}
	fmt.Println(vms)

	for _, val := range vms {
		name, err := val.Name()
		if err != nil {
			errExit(err)
		}
		id, err := val.ID()
		if err != nil {
			errExit(err)
		}
		fmt.Println(name, id)

		nics, err := val.ListVnics()
		if err != nil {
			errExit(err)
		}
		fmt.Printf("NICS: %v", nics)

		vm, err := vmm.GetVM(id)
		if err != nil {
			errExit(err)
		}
		fmt.Println(vm)
		aa, err := vm.GetSCSIControllers()
		if err != nil {
			errExit(err)
		}
		fmt.Println(aa)
		if len(aa) > 0 {
			for _, val := range aa {
				devs, err := val.AttachedDevices()
				if err != nil {
					errExit(err)
				}
				fmt.Println(devs)
				empty, err := val.EmptySlots()
				if err != nil {
					errExit(err)
				}
				fmt.Println(empty)
			}
		}
	}
}
