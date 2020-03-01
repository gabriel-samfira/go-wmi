package main

import (
	"fmt"
	"os"

	vm "go-wmi/virt/vm"
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
		fmt.Println(err)
		os.Exit(1)
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

		vm, err := vmm.GetVM(id)
		if err != nil {
			errExit(err)
		}
		fmt.Println(vm)
		aa, err := vm.MaybeCreateSCSIController()
		fmt.Println(aa, err)
	}
}
