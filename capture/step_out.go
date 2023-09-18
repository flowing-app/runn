package capture

type stepOut struct {
	Key     string      `json:"key"`
	Desc    string      `json:"desc"`
	Skipped bool        `json:"skipped"`
	Req     *stepOutReq `json:"req"`
	Res     *stepOutRes `json:"res"`
	Cond    []string    `json:"cond"`
	Err     string      `json:"err"`
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
