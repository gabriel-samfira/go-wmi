package utils

import (
	"crypto/rand"
	"fmt"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

// UUID4 returns a new UUID4
func UUID4() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// GetElementsAssociatedClass returns a list of WMI paths to class instances identified by instanceID
func GetElementsAssociatedClass(conn *wmi.WMI, className, instanceID string, extraQParams []wmi.Query) ([]string, error) {
	fields := []string{}
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "InstanceID",
				Value: fmt.Sprintf("%%%s%%", instanceID),
				Type:  wmi.Like},
		},
	}
	if extraQParams != nil && len(extraQParams) > 0 {
		qParams = append(qParams, extraQParams...)
	}
	results, err := conn.Gwmi(className, fields, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "Gwmi")
	}

	elem, err := results.Elements()
	if err != nil {
		return nil, errors.Wrap(err, "Elements")
	}

	ret := make([]string, len(elem))
	for idx, val := range elem {
		pth, err := val.Path()
		if err != nil {
			return nil, errors.Wrap(err, "Path")
		}
		ret[idx] = pth
	}
	return ret, nil
}

// RemoveResourceSettings removes a list of resource settings
func RemoveResourceSettings(svc *wmi.Result, resources []string) error {
	// RemoveResourceSettings
	jobPath := ole.VARIANT{}
	jobState, err := svc.Get("RemoveResourceSettings", resources, &jobPath)
	if err != nil {
		return errors.Wrap(err, "calling ModifyResourceSettings")
	}
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "waiting for job")
		}
	}
	return nil
}

// AddResourceSetting adds the resource settings to the specified VM
func AddResourceSetting(svc *wmi.Result, settingsData []string, vmPath string) ([]string, error) {
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

// GetResourceAllocSettings gets the resource allocation settings for the specified
// resoutce sub type and class.
func GetResourceAllocSettings(con *wmi.WMI, resourceSubType string, class string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "InstanceID",
				Value: "%%\\\\Default",
				Type:  wmi.Like,
			},
		},
	}
	if resourceSubType != "" {
		qParams = append(qParams, &wmi.AndQuery{
			wmi.QueryFields{
				Key:   "ResourceSubType",
				Value: resourceSubType,
				Type:  wmi.Equals,
			},
		})
	}
	settingsData, err := con.GetOne(class, []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "getting result")
	}
	return settingsData, nil
}
