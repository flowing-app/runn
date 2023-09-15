package capture

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type StepOut struct {
	Key     string      `json:"key"`
	Desc    string      `json:"desc"`
	Skipped bool        `json:"skipped"`
	Req     *StepOutReq `json:"req"`
	Res     *StepOutRes `json:"res"`
	Err     string      `json:"err"`
}

type StepOutReq struct {
	URI    string              `json:"URI"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

func NewStepOutReq(r *http.Request) (*StepOutReq, error) {
	url := r.URL.String()

	header := make(map[string][]string)
	for k, v := range r.Header {
		header[k] = v
	}

	var body []byte
	if r.Body != nil {
		rc, err := r.GetBody()
		if err != nil {
			return nil, fmt.Errorf("failed to GetBody: %w", err)
		}

		body, err = io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("failed to RaadAll: %w", err)
		}
		defer rc.Close()
	}

	return newStepOutReq(url, header, string(body)), nil
}

func newStepOutReq(URI string, header map[string][]string, body string) *StepOutReq {
	return &StepOutReq{
		URI:    URI,
		Header: header,
		Body:   body,
	}
}

type StepOutRes struct {
	Status string              `json:"status"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

func NewStepOutRes(r *http.Response) (*StepOutRes, error) {
	header := make(map[string][]string)
	for k, v := range r.Header {
		header[k] = v
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return newStepOutRes(r.Status, header, string(body)), nil

}

func newStepOutRes(status string, header map[string][]string, body string) *StepOutRes {
	return &StepOutRes{
		Status: status,
		Header: header,
		Body:   body,
	}
}
