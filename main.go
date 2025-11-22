package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"seat-killer/booker"
	"seat-killer/config"
	"seat-killer/mapper"
	"seat-killer/retry"
	"seat-killer/sso"
	"seat-killer/user"
)

const (
	// Total duration of the fallback window after the official booking time.
	fallbackWindow = 15 * time.Second
	// Interval between requests.
	requestInterval = 500 * time.Millisecond
)

func main() {
	log.Println("Starting Seat Killer...")

	// --- 1. Load Configs & Map ---
	//通过 user_info 加载当前用户信息结构体
	userInfo, err := config.LoadUserInfo("user_info.yml")
	if err != nil {
		log.Fatalf("Failed to load user_info.yml: %v", err)
	}
	// 通过 user_config 读取抢座任务信息，并返回 go 语言可读取的结构体
	seatCfg, err := config.LoadSeatConfig("user_config.yml")
	if err != nil {
		log.Fatalf("Failed to load user_config.yml: %v", err)
	}

	if _, err = mapper.LoadSeatMap("seat_report.txt"); err != nil {
		log.Fatalf("Failed to load seat map: %v", err)
	}
	log.Println("Configs and seat map loaded.")
	log.Printf("Loaded user config for SchoolID: %s", userInfo.SchoolID)

	// --- 2. Validate Credentials ---
	log.Println("Validating user credentials...")
	validationFunc := func() error {
		return sso.ValidateCredentials(userInfo.SchoolID, userInfo.Password)
	}
	if err := retry.WithRetry(validationFunc, 3, 2*time.Second); err != nil {
		log.Fatalf("Credential validation failed after multiple retries: %v. Please check your user_info.yml.", err)
	}
	log.Println("User credentials are valid.")

	// --- 3. Determine Today's Booking Task ---
	weekdayMap := map[time.Weekday]string{
		time.Sunday: "周日", time.Monday: "周一", time.Tuesday: "周二",
		time.Wednesday: "周三", time.Thursday: "周四", time.Friday: "周五",
		time.Saturday: "周六",
	}
	//通过调用时间函数返回值，匹配哈希表，得到今天是周几的中文
	todayWeekdayStr := weekdayMap[time.Now().Weekday()]
	//通过处理函数获取当前要请求的位置
	dayConfig, ok := seatCfg.WeekConfig[todayWeekdayStr]
	if !ok || !dayConfig.Enable || len(dayConfig.Seats) == 0 {
		log.Printf("Booking is not enabled for today (%s) or no seats configured. Exiting.", todayWeekdayStr)
		return
	}
	log.Printf("Found booking task for today (%s): Run at %d:%02d to book one of %d seat(s).",
		todayWeekdayStr, dayConfig.RunAtHour, dayConfig.RunAtMinute, len(dayConfig.Seats))

	bookingDayForLog := time.Now().AddDate(0, 0, 2)
	targetTime := time.Date(bookingDayForLog.Year(), bookingDayForLog.Month(), bookingDayForLog.Day(), dayConfig.BookStartHour, 0, 0, 0, time.Local)
	log.Printf("Task for SchoolID [%s]: Booking for %s, from %s for %d hours. Seats: %v",
		userInfo.SchoolID,
		targetTime.Format("2006-01-02"),
		targetTime.Format("15:04"),
		dayConfig.Duration,
		dayConfig.Seats)

	// --- 4. Define Time Windows ---
	now := time.Now()
	officialBookTime := time.Date(now.Year(), now.Month(), now.Day(), dayConfig.RunAtHour, dayConfig.RunAtMinute, 0, 0, time.Local)
	preemptTime := officialBookTime.Add(-time.Duration(seatCfg.Global.PreemptSeconds) * time.Second)
	fallbackEndTime := officialBookTime.Add(fallbackWindow)

	log.Printf("Attack Phase: %s -> %s (Primary Seat)", preemptTime.Format("15:04:05"), officialBookTime.Format("15:04:05"))
	log.Printf("Fallback Phase: %s -> %s (All Seats)", officialBookTime.Format("15:04:05"), fallbackEndTime.Format("15:04:05"))

	// --- 5. Wait for the first window ---
	if now.Before(preemptTime) {
		time.Sleep(preemptTime.Sub(now))
	}
	if time.Now().After(fallbackEndTime) {
		log.Println("Booking window has already passed. Exiting.")
		return
	}

	// --- 6. Login and Prepare ---
	log.Println("Booking window opened. Logging in...")
	var client *http.Client
	loginFunc := func() error {
		var loginErr error
		client, _, loginErr = sso.Login(userInfo.SchoolID, userInfo.Password)
		return loginErr
	}
	// Retry login for up to a minute to handle temporary service unavailability.
	// 20 attempts with a 3-second delay gives a ~1 minute window.
	if err := retry.WithRetry(loginFunc, 20, 3*time.Second); err != nil {
		log.Fatalf("Login failed after persistent retries for ~1 minute: %v", err)
	}
	loggedInUser, err := user.GetUserInfo(client)
	if err != nil {
		log.Fatalf("User info fetch failed: %v", err)
	}
	log.Printf("Logged in as SchoolID [%s] (UID: %s). Starting high-frequency requests...", userInfo.SchoolID, loggedInUser.UID)

	// --- 7. Execute Phased Booking ---
	if success, seat := executeBookingPhase(client, userInfo, loggedInUser, &dayConfig, preemptTime, officialBookTime, true); success {
		bookingDay := time.Now().AddDate(0, 0, 2)
		bookTime := time.Date(bookingDay.Year(), bookingDay.Month(), bookingDay.Day(), dayConfig.BookStartHour, 0, 0, 0, time.Local)
		log.Printf("BOOKING SUCCESSFUL for SchoolID [%s] in Attack Phase! Seat '%s' in room '%s' booked for %s from %s for %d hours.",
			userInfo.SchoolID,
			seat,
			dayConfig.Name,
			bookTime.Format("2006-01-02"),
			bookTime.Format("15:04"),
			dayConfig.Duration)
		return
	}
	if success, seat := executeBookingPhase(client, userInfo, loggedInUser, &dayConfig, officialBookTime, fallbackEndTime, false); success {
		bookingDay := time.Now().AddDate(0, 0, 2)
		bookTime := time.Date(bookingDay.Year(), bookingDay.Month(), bookingDay.Day(), dayConfig.BookStartHour, 0, 0, 0, time.Local)
		log.Printf("BOOKING SUCCESSFUL for SchoolID [%s] in Fallback Phase! Seat '%s' in room '%s' booked for %s from %s for %d hours.",
			userInfo.SchoolID,
			seat,
			dayConfig.Name,
			bookTime.Format("2006-01-02"),
			bookTime.Format("15:04"),
			dayConfig.Duration)
		return
	}

	log.Println("Seat Killer finished: all attempts failed within all windows.")
}

// executeBookingPhase runs the booking loop for a specific time window and seat strategy.
// Returns true if booking was successful.
func executeBookingPhase(client *http.Client, cfgUser *config.UserInfo, loggedInUser *user.UserInfo, dayCfg *config.DayConfig, start, end time.Time, primaryOnly bool) (bool, string) {
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	seatsToTry := dayCfg.Seats
	if primaryOnly {
		seatsToTry = []string{dayCfg.Seats[0]}
		log.Printf("--- Entering Attack Phase for SchoolID [%s]: Focusing on primary seat %s ---", cfgUser.SchoolID, seatsToTry[0])
	} else {
		log.Printf("--- Entering Fallback Phase for SchoolID [%s]: Trying all %d seats ---", cfgUser.SchoolID, len(seatsToTry))
	}

	bookingDay := time.Now().AddDate(0, 0, 2)

	for t := range ticker.C {
		if t.Before(start) {
			continue
		}
		if t.After(end) {
			return false, ""
		}

		for _, seatNum := range seatsToTry {
			seatID, err := mapper.GetSeatID(dayCfg.Name, seatNum)
			if err != nil {
				log.Printf("Cannot find seat '%s' in room '%s', skipping.", seatNum, dayCfg.Name)
				continue
			}

			bookTime := time.Date(bookingDay.Year(), bookingDay.Month(), bookingDay.Day(), dayCfg.BookStartHour, 0, 0, 0, time.Local)
			duration := time.Duration(dayCfg.Duration) * time.Hour

			log.Printf("Attempting to book for SchoolID [%s]: room '%s', seat '%s' (%d)", cfgUser.SchoolID, dayCfg.Name, seatNum, seatID)
			var result *booker.BookResponseData
			bookFunc := func() error {
				var bookErr error
				result, bookErr = booker.BookSeat(client, loggedInUser.UID, seatID, bookTime, duration)
				// If there's a booking error but the response indicates a non-retryable server message, wrap it.
				if bookErr == nil && !result.IsSuccess() && result.MESSAGE != "" {
					// Let's consider messages like "request too frequent" or "seat taken" as unretryable for the *immediate* retry.
					// The outer ticker loop will handle the next attempt after the interval.
					return retry.WrapUnretryable(fmt.Errorf("booking failed with server message: [%v] %s", result.CODE, result.MESSAGE))
				}
				return bookErr
			}

			if err := retry.WithRetry(bookFunc, 2, 100*time.Millisecond); err != nil {
				// Log the final error after retries, but don't stop the whole process.
				log.Printf("Booking attempt for seat %d failed after retries: %v", seatID, err)
				continue
			}

			log.Printf("Booking result for SchoolID [%s]: [%v] %s", cfgUser.SchoolID, result.CODE, result.MESSAGE)
			if result.IsSuccess() {
				return true, seatNum
			}
		}
	}
	return false, "" // Should not be reached if end time is handled correctly
}
