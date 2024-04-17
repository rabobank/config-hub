package util

func HasApplication(apps []string) bool {
	for _, app := range apps {
		if app == "application" {
			return true
		}
	}
	return false
}
