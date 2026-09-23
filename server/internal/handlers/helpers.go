package handlers

// asArray normalizes a nil slice to an empty one so JSON encodes [] instead
// of null. The frontend types every list as an array (e.g. Agent[]) and
// crashes on null (e.g. agents.length). GORM leaves slices nil when no rows
// match, so wrap every slice at the JSON boundary.
func asArray[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
