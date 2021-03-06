package engine

import(
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"github.com/google/go-cmp/cmp"
	"github.com/opwire/opwire-testa/lib/utils"
)

type SpecHandlerOptions interface {
}

type SpecHandler struct {
	invoker *HttpInvoker
}

func NewSpecHandler(opts SpecHandlerOptions) (e *SpecHandler, err error) {
	e = &SpecHandler{}
	e.invoker, err = NewHttpInvoker(nil)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *SpecHandler) Examine(testcase *TestCase) (*ExaminationResult, error) {
	if testcase == nil {
		return nil, fmt.Errorf("TestCase must not be nil")
	}

	result := &ExaminationResult{}

	// check if testcase is skipped
	if testcase.Skipped != nil && *testcase.Skipped == true {
		result.Status = "skipped"
		return result, nil
	}

	// make the testing request
	res, err := e.invoker.Do(testcase.Request)
	if err != nil {
		return nil, err
	}
	result.Response = res

	// matching with expectation
	errors := make(map[string]error, 0)
	expect := testcase.Expectation
	if expect != nil {
		_sc := expect.StatusCode
		if _sc != nil {
			if _sc.IsEqualTo != nil {
				if res.StatusCode != *_sc.IsEqualTo {
					errors["StatusCode"] = fmt.Errorf("Response StatusCode [%d] is not equal to expected value [%d]", res.StatusCode, *_sc.IsEqualTo)
				}
			}
			if _sc.BelongsTo != nil {
				if !utils.ContainsInt(_sc.BelongsTo, res.StatusCode) {
					errors["StatusCode"] = fmt.Errorf("Response StatusCode [%d] does not belong to expected list %v", res.StatusCode, _sc.BelongsTo)
				}
			}
		}
		_hs := expect.Headers
		if _hs != nil {
			if _hs.Items != nil {
				for _, item := range _hs.Items {
					headerVal := res.Header.Get(*item.Name)
					if item.IsEqualTo != nil && *item.IsEqualTo != headerVal {
						errors[fmt.Sprintf("Header[%s]", *item.Name)] = fmt.Errorf("Returned value: [%s] is mismatched with expected: [%s]", headerVal, *item.IsEqualTo)
					}
				}
			}
		}
		_eb := expect.Body
		if _eb != nil {
			if *_eb.HasFormat == "flat" {
				if _eb.IsEqualTo != nil {
					_rb := string(res.Body)
					if res.Body == nil || _rb != *_eb.IsEqualTo {
						errors["Body"] = fmt.Errorf("Response body is mismatched with expected content.\n    Received: %s\n    Expected: %s", _rb, *_eb.IsEqualTo)
					}
				}
			}
			if *_eb.HasFormat == "json" {
				var receivedJSON, expectedJSON interface{}
				next := true
				if (res.Body == nil) {
					errors["Body/receivedJSON"] = fmt.Errorf("Response body is empty (invalid JSON)")
					next = false
				} else if err := json.Unmarshal(res.Body, &receivedJSON); err != nil {
					errors["Body/receivedJSON"] = fmt.Errorf("Invalid response content: %s", err)
					next = false
				}
				if next && _eb.IsEqualTo != nil {
					if err := json.Unmarshal([]byte(*_eb.IsEqualTo), &expectedJSON); err != nil {
						errors["Body/expectedJSON"] = fmt.Errorf("Invalid expected content: %s", err)
						next = false
					}
					if next {
						if diff := cmp.Diff(expectedJSON, receivedJSON); diff != "" {
							errors["Body/IsEqualTo"] = fmt.Errorf("Body mismatch (-expected +received):\n%s", diff)
						}
					}
				}
				if next && _eb.Includes != nil {
					if err := json.Unmarshal([]byte(*_eb.Includes), &expectedJSON); err != nil {
						errors["Body/expectedJSON"] = fmt.Errorf("Invalid expected content: %s", err)
						next = false
					}
					if next {
						var r DiffReporter
						diff := cmp.Diff(expectedJSON, receivedJSON, cmp.Reporter(&r))
						if r.HasDiffs() {
							errors["Body/Includes"] = fmt.Errorf("Body mismatch (-expected +received):\n%s", diff)
						}
					}
				}
			}
		}
	}
	result.Errors = errors

	if len(errors) == 0 {
		result.Status = "ok"
	} else {
		result.Status = "error"
	}

	return result, nil
}

type TestSuite struct {
	TestCases []*TestCase `yaml:"testcases" json:"testcases"`
	Skipped *bool `yaml:"skipped,omitempty" json:"skipped"`
}

type TestCase struct {
	Title string `yaml:"title" json:"title"`
	Version *string `yaml:"version,omitempty" json:"version"`
	Request *HttpRequest `yaml:"request" json:"request"`
	Expectation *Expectation `yaml:"expectation" json:"expectation"`
	Skipped *bool `yaml:"skipped,omitempty" json:"skipped"`
	Tags []string `yaml:"tags,omitempty" json:"tags"`
	CreatedTime *string `yaml:"created-time,omitempty" json:"created-time"`
}

type Expectation struct {
	StatusCode *MeasureStatusCode `yaml:"status-code,omitempty" json:"status-code"`
	Headers *MeasureHeaders `yaml:"headers,omitempty" json:"headers"`
	Body *MeasureBody `yaml:"body,omitempty" json:"body"`
}

type MeasureStatusCode struct {
	IsEqualTo *int `yaml:"is-equal-to,omitempty" json:"is-equal-to"`
	BelongsTo []int `yaml:"belongs-to,omitempty" json:"belongs-to"`
}

type MeasureHeaders struct {
	Total *MeasureNumber `yaml:"total,omitempty" json:"total"`
	Items []MeasureHeader `yaml:"items,omitempty" json:"items"`
}

type MeasureNumber struct {
	IsEqualTo *int `yaml:"is-equal-to,omitempty" json:"is-equal-to"`
	IsLT *int `yaml:"is-lt,omitempty" json:"is-lt"`
	IsLTE *int `yaml:"is-lte,omitempty" json:"is-lte"`
	IsGT *int `yaml:"is-gt,omitempty" json:"is-gt"`
	IsGTE *int `yaml:"is-gte,omitempty" json:"is-gte"`
}

type MeasureHeader struct {
	Name *string `yaml:"name" json:"name"`
	IsEqualTo *string `yaml:"is-equal-to" json:"is-equal-to"`
}

type MeasureBody struct {
	HasFormat *string `yaml:"has-format,omitempty" json:"has-format"`
	Includes *string `yaml:"includes,omitempty" json:"includes"`
	IsEqualTo *string `yaml:"is-equal-to,omitempty" json:"is-equal-to"`
	MatchWith *string `yaml:"match-with,omitempty" json:"match-with"`
}

type ExaminationResult struct {
	Errors map[string]error
	Response *HttpResponse
	Status string
}

type DiffReporter struct {
	path  cmp.Path
	diffs []string
}

func (r *DiffReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

func (r *DiffReporter) Report(rs cmp.Result) {
	if !rs.Equal() {
		vx, vy := r.path.Last().Values()
		if !IsZero(vx) {
			r.diffs = append(r.diffs, fmt.Sprintf("%#v:\n\t-: %+v\n\t+: %+v\n", r.path, vx, vy))
		}
	}
}

func (r *DiffReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

func (r *DiffReporter) String() string {
	return strings.Join(r.diffs, "\n")
}

func (r *DiffReporter) HasDiffs() bool {
	return len(r.diffs) > 0
}

func (r *DiffReporter) GetDiffs() []string {
	return r.diffs
}

func IsZero(v reflect.Value) bool {
	return !v.IsValid() || reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}
