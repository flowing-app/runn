package runn

import "fmt"

type TrailType string

const (
	TrailTypeRunbook    TrailType = "runbook"
	TrailTypeStep       TrailType = "Step"
	TrailTypeBeforeFunc TrailType = "beforeFunc"
	TrailTypeAfterFunc  TrailType = "afterFunc"
)

type RunnerType string

const (
	RunnerTypeHTTP    RunnerType = "http"
	RunnerTypeDB      RunnerType = "db"
	RunnerTypeGRPC    RunnerType = "grpc"
	RunnerTypeCDP     RunnerType = "cdp"
	RunnerTypeSSH     RunnerType = "ssh"
	RunnerTypeExec    RunnerType = "exec"
	RunnerTypeTest    RunnerType = "test"
	RunnerTypeDump    RunnerType = "dump"
	RunnerTypeInclude RunnerType = "include"
	RunnerTypeBind    RunnerType = "bind"
)

// Trail - The trail of elements in the runbook at runtime
type Trail struct {
	Type           TrailType  `json:"type"`
	Desc           string     `json:"desc,omitempty"`
	RunbookID      string     `json:"id,omitempty"`
	RunbookPath    string     `json:"path,omitempty"`
	StepKey        string     `json:"Key,omitempty"`
	StepRunnerType RunnerType `json:"runner_type,omitempty"`
	StepRunnerKey  string     `json:"runner_key,omitempty"`
	FuncIndex      int        `json:"func_index,omitempty"`
}

type Trails []Trail

func (tr Trail) String() string {
	switch tr.Type {
	case TrailTypeRunbook:
		return fmt.Sprintf("runbook[%s]", tr.RunbookPath)
	case TrailTypeStep:
		return fmt.Sprintf("steps[%s]", tr.StepKey)
	case TrailTypeBeforeFunc:
		return fmt.Sprintf("beforeFunc[%d]", tr.FuncIndex)
	case TrailTypeAfterFunc:
		return fmt.Sprintf("afterFunc[%d]", tr.FuncIndex)
	default:
		return "invalid"
	}
}

func (trs Trails) toInterfaceSlice() []any {
	s := make([]any, len(trs))
	for i, v := range trs {
		s[i] = v
	}
	return s
}
