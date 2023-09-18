package capture

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

func (ss stepStore) getReq(key string) *StepOutReq {
	step := ss.get(key)
	if step == nil {
		return nil
	}

	v, ok := step[storeReqKey]
	if !ok {
		return nil
	}

	req, ok := v.(*StepOutReq)
	if !ok {
		return nil
	}

	return req
}

func (ss stepStore) getRes(key string) *StepOutRes {
	step := ss.get(key)
	if step == nil {
		return nil
	}

	v, ok := step[storeResKey]
	if !ok {
		return nil
	}

	res, ok := v.(*StepOutRes)
	if !ok {
		return nil
	}

	return res
}

func (ss stepStore) getCond(key string) string {
	step := ss.get(key)
	if step == nil {
		return ""
	}

	v, ok := step[storeCondKey]
	if !ok {
		return ""
	}

	cond, ok := v.(string)
	if !ok {
		return ""
	}

	return cond
}
