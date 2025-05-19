package timeutils

import "time"

// Возвращает время начала текущего месяца
func BeginOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func BeginOfNextMonth(t time.Time) time.Time {
	y := t.Year()
	m := t.Month()
	if m == 12 {
		m = 1
		y--
	}

	return time.Date(y, m, 0, 0, 0, 0, 0, time.UTC)
}
