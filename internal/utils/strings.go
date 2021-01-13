package utils

import "net/url"

func RedactedDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}

	if _, has := u.User.Password(); has {
		u.User = url.UserPassword(u.User.Username(), "xxxxx")
	}

	return u.String()
}

func ReverseStringSlice(numbers []string) []string {
	newNumbers := make([]string, 0, len(numbers))
	for i := len(numbers) - 1; i >= 0; i-- {
		newNumbers = append(newNumbers, numbers[i])
	}
	return newNumbers
}
