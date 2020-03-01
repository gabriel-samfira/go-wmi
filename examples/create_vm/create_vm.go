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

// CreateVM(name string, memoryMB int32, cpus int32, limitCPUFeatures bool, notes []string, generation GenerationType)
func main() {
	vmm, err := vm.NewVMManager()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer vmm.Release()

	vm, err := vmm.CreateVM("cucu", 3072, 3, false, []string{"ana", "are", "mere"}, vm.Generation2)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(vm)

	scsi, err := vm.GetOrCreateSCSIController()
	fmt.Println(scsi, err)
}
