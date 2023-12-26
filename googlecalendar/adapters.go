package googlecalendar

func listToStringSlice(src []interface{}) []string {
	dst := make([]string, 0, len(src))
	for _, s := range src {
		val, ok := s.(string)
		if !ok {
			val = ""
		}
		dst = append(dst, val)
	}
	return dst
}
