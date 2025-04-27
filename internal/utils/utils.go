package utils

import "time"

func ExtractMonthKey(dateStr string) (string, error) {
	var t time.Time
	var err error

	t, err = time.Parse(time.RFC3339, dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return "", err
		}
	}

	return t.Format("2006-01"), nil
}
