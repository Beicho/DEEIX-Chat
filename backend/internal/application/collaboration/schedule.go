package collaboration

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	scheduleTypeOnce   = "once"
	scheduleTypeDaily  = "daily"
	scheduleTypeWeekly = "weekly"
	scheduleTypeCron   = "cron"
)

func computeNextRunAt(scheduleType string, scheduleTime string, weekday int, cronExpression string, now time.Time) (time.Time, error) {
	switch normalizeScheduleType(scheduleType) {
	case scheduleTypeDaily:
		hour, minute, err := parseScheduleClock(scheduleTime)
		if err != nil {
			return time.Time{}, err
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next, nil
	case scheduleTypeWeekly:
		hour, minute, err := parseScheduleClock(scheduleTime)
		if err != nil {
			return time.Time{}, err
		}
		target := time.Weekday(weekday)
		if weekday < 0 || weekday > 6 {
			target = time.Monday
		}
		days := (int(target) - int(now.Weekday()) + 7) % 7
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()).AddDate(0, 0, days)
		if !next.After(now) {
			next = next.AddDate(0, 0, 7)
		}
		return next, nil
	case scheduleTypeCron:
		return nextFromFiveFieldCron(cronExpression, now)
	default:
		return time.Time{}, fmt.Errorf("unsupported schedule type")
	}
}

func normalizeScheduleType(value string) string {
	switch strings.TrimSpace(value) {
	case scheduleTypeDaily:
		return scheduleTypeDaily
	case scheduleTypeWeekly:
		return scheduleTypeWeekly
	case scheduleTypeCron:
		return scheduleTypeCron
	default:
		return scheduleTypeOnce
	}
}

func parseScheduleClock(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid schedule time")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid schedule hour")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid schedule minute")
	}
	return hour, minute, nil
}

func nextFromFiveFieldCron(expression string, now time.Time) (time.Time, error) {
	fields := strings.Fields(strings.TrimSpace(expression))
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("invalid cron expression")
	}
	minutes, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, err
	}
	hours, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, err
	}
	days, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, err
	}
	months, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, err
	}
	weekdays, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, err
	}

	cursor := now.Truncate(time.Minute).Add(time.Minute)
	deadline := cursor.AddDate(1, 0, 0)
	for cursor.Before(deadline) {
		if months[int(cursor.Month())] && days[cursor.Day()] && weekdays[int(cursor.Weekday())] && hours[cursor.Hour()] && minutes[cursor.Minute()] {
			return cursor, nil
		}
		cursor = cursor.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("cron expression has no next run within one year")
}

func parseCronField(field string, min int, max int) (map[int]bool, error) {
	result := map[int]bool{}
	trimmed := strings.TrimSpace(field)
	if trimmed == "*" {
		for value := min; value <= max; value++ {
			result[value] = true
		}
		return result, nil
	}
	for _, part := range strings.Split(trimmed, ",") {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value < min || value > max {
			return nil, fmt.Errorf("invalid cron field")
		}
		result[value] = true
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("invalid cron field")
	}
	return result, nil
}
