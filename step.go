package runn

import "errors"

type Step struct {
	Key           string
	runnerKey     string
	desc          string
	ifCond        string
	loop          *Loop
	httpRunner    *httpRunner
	httpRequest   map[string]any
	dbRunner      *dbRunner
	dbQuery       map[string]any
	grpcRunner    *grpcRunner
	grpcRequest   map[string]any
	cdpRunner     *cdpRunner
	cdpActions    map[string]any
	sshRunner     *sshRunner
	sshCommand    map[string]any
	execRunner    *execRunner
	execCommand   map[string]any
	testRunner    *testRunner
	TestCond      string
	dumpRunner    *dumpRunner
	dumpRequest   *dumpRequest
	bindRunner    *bindRunner
	bindCond      map[string]string
	includeRunner *includeRunner
	includeConfig *includeConfig
	// operator related to Step
	Parent *operator
	debug  bool
	result *StepResult
}

func newStep(key string, parent *operator) *Step {
	return &Step{Key: key, Parent: parent, debug: parent.debug}
}

func (s *Step) generateTrail() Trail {
	tr := Trail{
		Type:          TrailTypeStep,
		Desc:          s.desc,
		StepKey:       s.Key,
		StepRunnerKey: s.runnerKey,
	}
	switch {
	case s.httpRunner != nil && s.httpRequest != nil:
		tr.StepRunnerType = RunnerTypeHTTP
	case s.dbRunner != nil && s.dbQuery != nil:
		tr.StepRunnerType = RunnerTypeDB
	case s.grpcRunner != nil && s.grpcRequest != nil:
		tr.StepRunnerType = RunnerTypeGRPC
	case s.cdpRunner != nil && s.cdpActions != nil:
		tr.StepRunnerType = RunnerTypeCDP
	case s.sshRunner != nil && s.sshCommand != nil:
		tr.StepRunnerType = RunnerTypeSSH
	case s.execRunner != nil && s.execCommand != nil:
		tr.StepRunnerType = RunnerTypeExec
	case s.includeRunner != nil && s.includeConfig != nil:
		tr.StepRunnerType = RunnerTypeInclude
	case s.dumpRunner != nil && s.dumpRequest != nil:
		tr.StepRunnerType = RunnerTypeDump
	case s.bindRunner != nil && s.bindCond != nil:
		tr.StepRunnerType = RunnerTypeBind
	case s.testRunner != nil && s.TestCond != "":
		tr.StepRunnerType = RunnerTypeTest
	}

	return tr
}

func (s *Step) trails() Trails {
	var trs Trails
	if s.Parent != nil {
		trs = s.Parent.trails()
	}
	trs = append(trs, s.generateTrail())
	return trs
}

func (s *Step) setResult(err error) {
	if s.result != nil {
		panic("duplicate record of Step results")
	}
	var runResult *RunResult
	if s.includeRunner != nil {
		runResult = s.includeRunner.runResult
	}
	if errors.Is(errStepSkiped, err) {
		s.result = &StepResult{Key: s.Key, Desc: s.desc, Skipped: true, Err: nil, IncludedRunResult: runResult}
		return
	}
	s.result = &StepResult{Key: s.Key, Desc: s.desc, Skipped: false, Err: err, IncludedRunResult: runResult}
}

func (s *Step) clearResult() {
	s.result = nil
}
