package capture

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const (
	stepStoreReqKey  = "req"
	stepStoreResKey  = "res"
	stepStoreCondKey = "cond"
)

type stepStore map[string]map[string]any

func (ss stepStore) get(step string) map[string]any {
	v, ok := ss[step]
	if !ok {
		return nil
	}
	return v
}

func (ss stepStore) getReq(step string) *stepOutReq {
	s := ss.get(step)
	if s == nil {
		return nil
	}

	v, ok := s[stepStoreReqKey]
	if !ok {
		return nil
	}

	req, ok := v.(*stepOutReq)
	if !ok {
		return nil
	}

	return req
}

func (ss stepStore) saveReq(step string, req *http.Request) error {
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

	ss[step][stepStoreReqKey] = newStepOutReq(url, header, string(body))

	return nil
}

func (ss stepStore) getRes(step string) *stepOutRes {
	s := ss.get(step)
	if s == nil {
		return nil
	}

	v, ok := s[stepStoreResKey]
	if !ok {
		return nil
	}

	res, ok := v.(*stepOutRes)
	if !ok {
		return nil
	}

	return res
}

func (ss stepStore) saveRes(step string, res *http.Response) error {
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

	ss[step][stepStoreResKey] = newStepOutRes(status, header, string(body))

	return nil
}

func (ss stepStore) getCond(step string) string {
	s := ss.get(step)
	if s == nil {
		return ""
	}

	v, ok := s[stepStoreCondKey]
	if !ok {
		return ""
	}

	cond, ok := v.(string)
	if !ok {
		return ""
	}

	return cond
}

func (ss stepStore) saveCond(key, cond string) {
	ss[key][stepStoreCondKey] = cond
}
