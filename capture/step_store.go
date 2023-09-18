package capture

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	storeReqKey  = "req"
	storeResKey  = "res"
	storeCondKey = "cond"
)

type stepStore map[string]map[string]any

func (ss stepStore) get(key string) map[string]any {
	v, ok := ss[key]
	if !ok {
		return nil
	}
	return v
}

func (ss stepStore) getReq(key string) *stepOutReq {
	step := ss.get(key)
	if step == nil {
		return nil
	}

	v, ok := step[storeReqKey]
	if !ok {
		return nil
	}

	req, ok := v.(*stepOutReq)
	if !ok {
		return nil
	}

	return req
}

func (ss stepStore) saveReq(key string, req *http.Request) error {
	url := req.URL.String()

	header := make(map[string][]string)
	for k, v := range req.Header {
		header[k] = v
	}

	var body []byte
	if req.Body != nil {
		rc, err := req.GetBody()
		if err != nil {
			return fmt.Errorf("failed to GetBody: %w", err)
		}

		body, err = io.ReadAll(rc)
		if err != nil {
			return fmt.Errorf("failed to RaadAll: %w", err)
		}
		defer rc.Close()
	}

	ss[key][storeReqKey] = newStepOutReq(url, header, string(body))

	return nil
}

func (ss stepStore) getRes(key string) *stepOutRes {
	step := ss.get(key)
	if step == nil {
		return nil
	}

	v, ok := step[storeResKey]
	if !ok {
		return nil
	}

	res, ok := v.(*stepOutRes)
	if !ok {
		return nil
	}

	return res
}

func (ss stepStore) saveRes(key string, res *http.Response) error {
	status := strconv.Itoa(res.StatusCode)

	header := make(map[string][]string)
	for k, v := range res.Header {
		header[k] = v
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	defer res.Body.Close()
	res.Body = io.NopCloser(bytes.NewBuffer(body))

	ss[key][storeResKey] = newStepOutRes(status, header, string(body))

	return nil
}

func (ss stepStore) getCond(key string) []string {
	step := ss.get(key)
	if step == nil {
		return nil
	}

	v, ok := step[storeCondKey]
	if !ok {
		return nil
	}

	cond, ok := v.([]string)
	if !ok {
		return nil
	}

	return cond
}

func (ss stepStore) saveCond(key string, cond string) {
	cond = strings.ReplaceAll(cond, "\n", " ")
	conds := strings.Split(cond, "&&")
	for i, c := range conds {
		conds[i] = strings.Trim(c, " ")
	}
	ss[key][storeCondKey] = conds
}
