package wmi

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	// "github.com/go-ole/go-ole/oleutil"

	// "github.com/gabriel-samfira/go-wmi/wmi"
)

// PathParser contains the parsed fields of a __PATH
type PathParser struct {
	// Server represents the server on which this query should be run
	Server string
	// Namespace represents the namespace in which to run the query
	Namespace string
	// Class represents the class against which to run the query
	Class string
	// Params is a map of parameters to filter
	Params map[string]string

	conn *WMI
}

var pathRegexp = regexp.MustCompile(`\\\\(?P<server>[a-zA-Z0-9-.]+)\\(?P<namespace>[a-zA-Z0-9\\]+):(?P<class>[a-zA-Z_]+)[.]?(?P<params>.*)?`)
var requiredFields = []string{
	"server",
	"namespace",
	"class",
}

func validateResult(result map[string]string) error {
	for _, val := range requiredFields {
		if _, ok := result[val]; !ok {
			return fmt.Errorf("Could not find reuired field: %s", val)
		}
	}
	return nil
}

func parsePath(path string) map[string]string {
	matches := pathRegexp.FindStringSubmatch(path)
	result := make(map[string]string)
	for i, val := range pathRegexp.SubexpNames() {
		if val == "" {
			continue
		}
		result[val] = matches[i]
	}
	return result
}

func parseParams(params string) (map[string]string, error) {
	s := strings.Split(params, ",")
	result := make(map[string]string)
	for _, v := range s {
		p := strings.Split(v, "=")
		if len(p) != 2 {
			return map[string]string{}, fmt.Errorf("Invalid parameters format: %s", params)
		}
		key, val := p[0], p[1]
		result[key] = strings.Trim(val, `"`)
	}
	return result, nil
}

func (p *PathParser) QueryParams() []WMIQuery {
	q := []WMIQuery{}
	if len(p.Params) > 0 {
		for key, val := range p.Params {
			q = append(q, &WMIAndQuery{QueryFields{Key: key, Type: Equals, Value: val}})
		}
	}
	return q
}

func NewPathParser(path string) (*PathParser, error) {
	result := parsePath(path)
	err := validateResult(result)
	if err != nil {
		return nil, err
	}
	params, err := parseParams(result["params"])
	if err != nil {
		return nil, err
	}
	return &PathParser{
		Server:    result["server"],
		Namespace: result["namespace"],
		Class:     result["class"],
		Params:    params,
	}, nil
}

// JobState represents a WMI job that was run. This class exposes a subset
// of the information available in CIM_ConcreteJob
// https://msdn.microsoft.com/en-us/library/cc136808%28v=vs.85%29.aspx
type JobState struct {
	Name             string
	Description      string
	DetailedStatus   string
	ElementName      string
	ErrorCode        int
	ErrorDescription string
	InstanceID       string
	JobRunTimes      int
	JobState         int
	JobStatus        string
	JobType          int
}

func populateJobData(j *WMIResult) (JobState, error) {
	job := &JobState{}
	valuePtr := reflect.ValueOf(job)
	v := reflect.Indirect(valuePtr)

	for i := 0; i < v.NumField(); i++ {
		field := valuePtr.Elem().Field(i)
		name := v.Type().Field(i).Name
		kind := v.Type().Field(i).Type.Kind()
		res, err := j.GetProperty(name)

		if err != nil {
			return JobState{}, err
		}
		jobFieldValue := res.Value()
		if jobFieldValue == nil {
			continue
		}
		switch kind {
		case reflect.Int:
			if v, ok := jobFieldValue.(int32); !ok {
				return JobState{}, fmt.Errorf("Invalid return value for %s: %T", name, jobFieldValue)
			} else {
				field.SetInt(int64(v))
			}
		case reflect.String:
			if v, ok := jobFieldValue.(string); !ok {
				return JobState{}, fmt.Errorf("Invalid return value for %s: %T", name, jobFieldValue)
			} else {
				field.SetString(v)
			}
		}
	}
	return *job, nil
}

func NewJobState(path string) (JobState, error) {
	connectData, err := NewPathParser(path)
	if err != nil {
		return JobState{}, err
	}
	if strings.HasSuffix(connectData.Class, "_ConcreteJob") == false {
		return JobState{}, fmt.Errorf("Path is not a valid CIM_ConcreteJob. Got: %s", connectData.Class)
	}
	w, err := NewConnection(connectData.Server, connectData.Namespace)
	if err != nil {
		return JobState{}, err
	}
	defer w.Close()
	jobData, err := w.GetOne(connectData.Class, []string{}, connectData.QueryParams())
	if err != nil {
		return JobState{}, err
	}
	defer jobData.Release()
	return populateJobData(jobData)
}

func WaitForJob(jobPath string) error {
	for {
		jobData, err := NewJobState(jobPath)
		if err != nil {
			return err
		}
		if jobData.JobState == WMI_JOB_STATE_RUNNING {
			continue
		}
		if jobData.JobState != WMI_JOB_STATE_COMPLETED {
			return fmt.Errorf("Job failed: %s (%d)", jobData.ErrorDescription, jobData.ErrorCode)
		}
	}
	return nil
}
