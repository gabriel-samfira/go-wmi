package wmi

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	// "github.com/go-ole/go-ole"
	// "github.com/go-ole/go-ole/oleutil"
	// "github.com/gabriel-samfira/go-wmi/wmi"
)

// Location contains the parsed fields of a __PATH
type Location struct {
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

// Close will close the WMI connection for this location
func (w *Location) Close() {
	w.conn.Close()
}

// GetResult wil return a Result for this Location
func (w *Location) GetResult() (*Result, error) {
	result, err := w.conn.GetOne(w.Class, []string{}, w.QueryParams())
	if err != nil {
		return nil, err
	}
	return result, nil
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

// QueryParams returns a []Query from the params present in the
// location string
func (w *Location) QueryParams() []Query {
	q := []Query{}
	if len(w.Params) > 0 {
		for key, val := range w.Params {
			q = append(q, &AndQuery{QueryFields{Key: key, Type: Equals, Value: val}})
		}
	}
	return q
}

// NewLocation returns a *Location object
func NewLocation(path string) (*Location, error) {
	result := parsePath(path)
	err := validateResult(result)
	if err != nil {
		return nil, err
	}
	params, err := parseParams(result["params"])
	if err != nil {
		return nil, err
	}
	w, err := NewConnection(result["server"], result["namespace"])
	if err != nil {
		return nil, err
	}
	return &Location{
		Server:    result["server"],
		Namespace: result["namespace"],
		Class:     result["class"],
		Params:    params,
		conn:      w,
	}, nil
}

// JobState represents a WMI job that was run. This type exposes a subset
// of the information available in CIM_ConcreteJob
// https://msdn.microsoft.com/en-us/library/cc136808%28v=vs.85%29.aspx
type JobState struct {
	Name             string
	Description      string
	ElementName      string
	ErrorCode        int32
	ErrorDescription string
	InstanceID       string
	JobRunTimes      int32
	JobState         int32
	JobStatus        string
	JobType          int32
}

// PopulateStruct populates the fields of the supplied struct
// with values received form a Result. Care must be taken when
// declaring the struct. It must match the types returned by WMI.
func PopulateStruct(j *Result, s interface{}) (err error) {
	var name string
	var fieldType interface{}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Invalid field type (%T) for %s: %s", fieldType, name, r)
		}
	}()
	valuePtr := reflect.ValueOf(s)
	elem := valuePtr.Elem()
	typeOfElem := elem.Type()

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		if typeOfElem.Field(i).Tag.Get("tag") == "ignore" {
			continue
		}
		name = typeOfElem.Field(i).Name
		fieldType = field.Interface()
		res, err := j.GetProperty(name)
		if err != nil {
			return fmt.Errorf("Failed to get property %s: %s", name, err)
		}

		wmiFieldValue := res.Value()
		if wmiFieldValue == nil {
			continue
		}

		var fieldValue interface{}
		switch field.Interface().(type) {
		case []uint16:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]uint16, len(val))
				for k, v := range val {
					asString[k] = v.(uint16)
				}
				fieldValue = asString
			}
		case []string:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]string, len(val))
				for k, v := range val {
					asString[k] = v.(string)
				}
				fieldValue = asString
			}
		case []uint32:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]uint32, len(val))
				for k, v := range val {
					asString[k] = v.(uint32)
				}
				fieldValue = asString
			}
		case []int32:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]int32, len(val))
				for k, v := range val {
					asString[k] = v.(int32)
				}
				fieldValue = asString
			}
		case []int64:
			if c := res.ToArray(); c != nil {
				val := c.ToValueArray()
				asString := make([]int64, len(val))
				for k, v := range val {
					asString[k] = v.(int64)
				}
				fieldValue = asString
			}
		default:
			fieldValue = wmiFieldValue
		}

		v := reflect.ValueOf(fieldValue)
		if v.Kind() != field.Kind() {
			return fmt.Errorf("Invalid type returned by query for field %s (%v): %v", name, v.Kind(), field.Kind())
		}
		if field.CanSet() {
			field.Set(v)
		}
	}
	return nil
}

// NewJobState returns a new Jobstate, given a path
func NewJobState(path string) (JobState, error) {
	conn, err := NewLocation(path)
	if err != nil {
		return JobState{}, err
	}
	defer conn.Close()
	// This may blow up. In theory, both CIM_ConcreteJob and Msvm_Concrete job will
	// work with this. Also, anything that inherits CIM_ConctreteJob will also work.
	// TODO: Make this more robust
	if strings.HasSuffix(conn.Class, "_ConcreteJob") == false {
		return JobState{}, fmt.Errorf("Path is not a valid ConcreteJob. Got: %s", conn.Class)
	}

	jobData, err := conn.GetResult()
	if err != nil {
		return JobState{}, err
	}

	j := JobState{}
	err = PopulateStruct(jobData, &j)
	if err != nil {
		return JobState{}, err
	}
	return j, nil
}

// WaitForJob will wait for a WMI job to complete
func WaitForJob(jobPath string) error {
	for {
		jobData, err := NewJobState(jobPath)
		if err != nil {
			return err
		}
		if jobData.JobState == JobStatusRunning {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if jobData.JobState != JobStateCompleted {
			return fmt.Errorf("Job failed: %s (%d)", jobData.ErrorDescription, jobData.ErrorCode)
		}
		break
	}
	return nil
}
