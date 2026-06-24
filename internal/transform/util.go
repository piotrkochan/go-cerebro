package transform

func getNested(m map[string]any, path ...string) any {
	cur := any(m)
	for _, p := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[p]
	}
	return cur
}

func getNumber(m map[string]any, path ...string) (float64, bool) {
	v := getNested(m, path...)
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}
