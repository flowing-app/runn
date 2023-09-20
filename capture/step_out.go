package capture

import "strings"

type stepOut struct {
	BookPath string      `json:"bookPath"`
	Key      string      `json:"key"`
	Desc     string      `json:"desc"`
	Skipped  bool        `json:"skipped"`
	Req      *stepOutReq `json:"req"`
	Res      *stepOutRes `json:"res"`
	Cond     []string    `json:"cond"`
	RawCond  string      `json:"rawCond"`
	Err      string      `json:"err"`
}

type stepOutReq struct {
	URL    string              `json:"URL"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

func newStepOutReq(URI string, header map[string][]string, body string) *stepOutReq {
	return &stepOutReq{
		URL:    URI,
		Header: header,
		Body:   body,
	}
}

type stepOutRes struct {
	Status string              `json:"status"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

func newStepOutRes(status string, header map[string][]string, body string) *stepOutRes {
	return &stepOutRes{
		Status: status,
		Header: header,
		Body:   body,
	}
}

func (so *stepOut) setCond(cond string) {
	replaced := strings.ReplaceAll(cond, "\n", " ")
	conds := strings.Split(replaced, "&&")
	for i, c := range conds {
		conds[i] = strings.Trim(c, " ")
	}

	so.Cond = conds
	so.RawCond = cond
}
