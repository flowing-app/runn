package runn

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/goccy/go-json"
	"google.golang.org/grpc/status"
)

var _ Capturer = (*CmdOutJson)(nil)

type CmdOutJson struct {
	out io.Writer
}

func NewCmdOutJson(out io.Writer) *CmdOutJson {
	return &CmdOutJson{
		out: out,
	}
}

type StepResultOut struct {
	Key     string `json:"key"`
	Desc    string `json:"desc"`
	Skipped bool   `json:"skipped"`
	Err     string `json:"err"`
}

func (d *CmdOutJson) CaptureStart(trs Trails, bookPath, desc string) {}
func (d *CmdOutJson) CaptureResult(trs Trails, result *RunResult)    {}
func (d *CmdOutJson) CaptureEnd(trs Trails, bookPath, desc string)   {}

func (d *CmdOutJson) CaptureStepStart(step *Step) {}
func (d *CmdOutJson) CaptureStepEnd(step *Step) {
	r := step.result
	o := StepResultOut{
		Key:     r.Key,
		Desc:    r.Desc,
		Skipped: r.Skipped,
	}
	if r.Err != nil {
		err := errors.Unwrap(r.Err).Error()
		err = strings.ReplaceAll(err, "\n", "\\n")
		err = strings.ReplaceAll(err, "\"", "\\\"")
		o.Err = err
	}

	b, _ := json.Marshal(o)
	fmt.Fprintf(d.out, "%s\n", b)
}

func (d *CmdOutJson) CaptureHTTPRequest(name string, req *http.Request)                  {}
func (d *CmdOutJson) CaptureHTTPResponse(name string, res *http.Response)                {}
func (d *CmdOutJson) CaptureGRPCStart(name string, typ GRPCType, service, method string) {}
func (d *CmdOutJson) CaptureGRPCRequestHeaders(h map[string][]string)                    {}
func (d *CmdOutJson) CaptureGRPCRequestMessage(m map[string]any)                         {}
func (d *CmdOutJson) CaptureGRPCResponseStatus(s *status.Status)                         {}
func (d *CmdOutJson) CaptureGRPCResponseHeaders(h map[string][]string)                   {}
func (d *CmdOutJson) CaptureGRPCResponseMessage(m map[string]any)                        {}
func (d *CmdOutJson) CaptureGRPCResponseTrailers(t map[string][]string)                  {}
func (d *CmdOutJson) CaptureGRPCClientClose()                                            {}
func (d *CmdOutJson) CaptureGRPCEnd(name string, typ GRPCType, service, method string)   {}
func (d *CmdOutJson) CaptureCDPStart(name string)                                        {}
func (d *CmdOutJson) CaptureCDPAction(a CDPAction)                                       {}
func (d *CmdOutJson) CaptureCDPResponse(a CDPAction, res map[string]any)                 {}
func (d *CmdOutJson) CaptureCDPEnd(name string)                                          {}
func (d *CmdOutJson) CaptureSSHCommand(command string)                                   {}
func (d *CmdOutJson) CaptureSSHStdout(stdout string)                                     {}
func (d *CmdOutJson) CaptureSSHStderr(stderr string)                                     {}
func (d *CmdOutJson) CaptureDBStatement(name string, stmt string)                        {}
func (d *CmdOutJson) CaptureDBResponse(name string, res *DBResponse)                     {}
func (d *CmdOutJson) CaptureExecCommand(command string)                                  {}
func (d *CmdOutJson) CaptureExecStdin(stdin string)                                      {}
func (d *CmdOutJson) CaptureExecStdout(stdout string)                                    {}
func (d *CmdOutJson) CaptureExecStderr(stderr string)                                    {}
func (d *CmdOutJson) SetCurrentTrails(trs Trails)                                        {}
func (d *CmdOutJson) Errs() error {
	return nil
}
