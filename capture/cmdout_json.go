package capture

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/k1LoW/runn"

	"github.com/goccy/go-json"
	"google.golang.org/grpc/status"
)

var _ runn.Capturer = (*CmdOutJson)(nil)

type CmdOutJson struct {
	out   io.Writer
	store stepStore
}

func NewCmdOutJson(out io.Writer) *CmdOutJson {
	return &CmdOutJson{
		out:   out,
		store: make(stepStore),
	}
}

func (d *CmdOutJson) CaptureStart(trs runn.Trails, bookPath, desc string)   {}
func (d *CmdOutJson) CaptureResult(trs runn.Trails, result *runn.RunResult) {}
func (d *CmdOutJson) CaptureEnd(trs runn.Trails, bookPath, desc string)     {}

func (d *CmdOutJson) CaptureStepStart(step *runn.Step) {
	d.store[step.Key] = make(map[string]any)
	d.store.saveCond(step.Key, step.TestCond)
	d.store.saveBookPath(step.Key, step.Parent.BookPath())
}
func (d *CmdOutJson) CaptureStepEnd(result *runn.StepResult) {
	o := stepOut{
		Key:      result.Key,
		Desc:     result.Desc,
		Skipped:  result.Skipped,
		Req:      d.store.getReq(result.Key),
		Res:      d.store.getRes(result.Key),
		BookPath: d.store.getBookPath(result.Key),
	}
	o.setCond(d.store.getCond(result.Key))

	if result.Err != nil {
		o.Err = errors.Unwrap(result.Err).Error()
	}

	fmtStepOut(&o)

	b, _ := json.MarshalIndent(o, "", "  ")
	fmt.Fprintf(d.out, "%s\n", b)
}

func fmtStepOut(out *stepOut) {
	out.Cond = fmtEscapeds(out.Cond)
	out.RawCond = fmtEscaped(out.RawCond)
	out.Err = fmtEscaped(out.Err)
}

func (d *CmdOutJson) CaptureHTTPRequest(name string, req *http.Request, s *runn.Step) {
	if _, ok := d.store[s.Key]; !ok {
		panic(fmt.Sprintf("step '%s' is not inittied", s.Key))
	}
	if err := d.store.saveReq(s.Key, req); err != nil {
		panic(fmt.Sprintf("failed to save request: %v", err))
	}
}
func (d *CmdOutJson) CaptureHTTPResponse(name string, res *http.Response, s *runn.Step) {
	if _, ok := d.store[s.Key]; !ok {
		panic(fmt.Sprintf("step '%s' is not inittied", s.Key))
	}
	if err := d.store.saveRes(s.Key, res); err != nil {
		panic(fmt.Sprintf("failed to save response: %v", err))
	}
}

func (d *CmdOutJson) CaptureGRPCStart(name string, typ runn.GRPCType, service, method string) {}
func (d *CmdOutJson) CaptureGRPCRequestHeaders(h map[string][]string)                         {}
func (d *CmdOutJson) CaptureGRPCRequestMessage(m map[string]any)                              {}
func (d *CmdOutJson) CaptureGRPCResponseStatus(s *status.Status)                              {}
func (d *CmdOutJson) CaptureGRPCResponseHeaders(h map[string][]string)                        {}
func (d *CmdOutJson) CaptureGRPCResponseMessage(m map[string]any)                             {}
func (d *CmdOutJson) CaptureGRPCResponseTrailers(t map[string][]string)                       {}
func (d *CmdOutJson) CaptureGRPCClientClose()                                                 {}
func (d *CmdOutJson) CaptureGRPCEnd(name string, typ runn.GRPCType, service, method string)   {}
func (d *CmdOutJson) CaptureCDPStart(name string)                                             {}
func (d *CmdOutJson) CaptureCDPAction(a runn.CDPAction)                                       {}
func (d *CmdOutJson) CaptureCDPResponse(a runn.CDPAction, res map[string]any)                 {}
func (d *CmdOutJson) CaptureCDPEnd(name string)                                               {}
func (d *CmdOutJson) CaptureSSHCommand(command string)                                        {}
func (d *CmdOutJson) CaptureSSHStdout(stdout string)                                          {}
func (d *CmdOutJson) CaptureSSHStderr(stderr string)                                          {}
func (d *CmdOutJson) CaptureDBStatement(name string, stmt string)                             {}
func (d *CmdOutJson) CaptureDBResponse(name string, res *runn.DBResponse)                     {}
func (d *CmdOutJson) CaptureExecCommand(command string)                                       {}
func (d *CmdOutJson) CaptureExecStdin(stdin string)                                           {}
func (d *CmdOutJson) CaptureExecStdout(stdout string)                                         {}
func (d *CmdOutJson) CaptureExecStderr(stderr string)                                         {}
func (d *CmdOutJson) SetCurrentTrails(trs runn.Trails)                                        {}
func (d *CmdOutJson) Errs() error {
	return nil
}

func fmtEscaped(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func fmtEscapeds(ss []string) []string {
	fmted := make([]string, len(ss))
	for i, s := range ss {
		fmted[i] = fmtEscaped(s)
	}
	return fmted
}
