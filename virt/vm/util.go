package vm

import (
	"fmt"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

func addResourceSetting(svc *wmi.Result, settingsData []string, vmPath string) ([]string, error) {
	jobPath := ole.VARIANT{}
	resultingSystem := ole.VARIANT{}
	jobState, err := svc.Get("AddResourceSettings", vmPath, settingsData, &resultingSystem, &jobPath)
	if err != nil {
		return nil, errors.Wrap(err, "calling ModifyResourceSettings")
	}

	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return nil, errors.Wrap(err, "waiting for job")
		}
	}
	safeArrayConversion := resultingSystem.ToArray()
	valArray := safeArrayConversion.ToValueArray()
	if len(valArray) == 0 {
		return nil, fmt.Errorf("no resource in resultingSystem value")
	}
	resultingSystems := make([]string, len(valArray))
	for idx, val := range valArray {
		resultingSystems[idx] = val.(string)
	}
	return resultingSystems, nil
}

func getResourceAllocSettings(con *wmi.WMI, resourceSubType string, class string) (*wmi.Result, error) {
	if class == "" {
		class = ResourceAllocSettingDataClass
	}

	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: resourceSubType,
				Type:  wmi.Equals,
			},
		},
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "InstanceID",
				Value: "%\\\\Default",
				Type:  wmi.Like,
			},
		},
	}
	settingsDataResults, err := con.Gwmi(class, []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("getting %s", class))
	}
	settingsData, err := settingsDataResults.ItemAtIndex(0)
	if err != nil {
		return nil, errors.Wrap(err, "getting result")
	}
	return settingsData, nil
}
