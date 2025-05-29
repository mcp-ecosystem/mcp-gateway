package utils

func FirstNonEmpty(str1, str2 string) string {
	if str1 != "" {
		return str1
	}
	if str2 != "" {
		return str2
	}
	return ""
}
