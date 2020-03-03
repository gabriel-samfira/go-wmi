package main

import (
	"flag"
	"fmt"
	"os"

	vm "github.com/gabriel-samfira/go-wmi/virt/vm"
)

func errExit(err error) {
	fmt.Println(err)
	os.Exit(1)
}

// CreateVM(name string, memoryMB int32, cpus int32, limitCPUFeatures bool, notes []string, generation GenerationType)
func main() {
	vhdxPath := flag.String("vhdx", "", "path to VHDX")
	flag.Parse()
	if *vhdxPath == "" {
		fmt.Println("please specify VHDX path")
		os.Exit(1)
	}

	if _, err := os.Stat(*vhdxPath); err != nil {
		fmt.Printf("failed to stat VHDX: %s\n", *vhdxPath)
		os.Exit(1)
	}

	vmm, err := vm.NewVMManager()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer vmm.Release()

	virtualMachine, err := vmm.CreateVM("cucu", 3072, 3, false, []string{"ana", "are", "mere"}, vm.Generation2, false)
	if err != nil {
		errExit(err)
	}
	fmt.Println(virtualMachine)

	scsi, err := virtualMachine.GetSCSIControllers()
	if err != nil {
		errExit(err)
	}
	fmt.Println(scsi)
	if len(scsi) == 0 {
		fmt.Println("Creating new SCSI controller")
		_, err = virtualMachine.CreateNewSCSIController()

	}
	scsi, err = virtualMachine.GetSCSIControllers()
	if err != nil {
		errExit(err)
	}
	for _, val := range scsi {
		fmt.Printf("SCSI controller is: %v\n", val)
	}

	ctrl := scsi[0]
	res, err := ctrl.AttachDrive(*vhdxPath, vm.DiskDrive)
	if err != nil {
		errExit(err)
	}
	fmt.Println(res)
}
