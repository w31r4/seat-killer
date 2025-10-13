package main

import (
	"log"
	"time"

	"seat-killer/booker"
	"seat-killer/config"
	"seat-killer/mapper"
	"seat-killer/sso"
	"seat-killer/user"
)

const (
	// Keep trying for 2 minutes after the official booking time.
	windowMinutes = 2
	// Interval between requests.
	requestInterval = 500 * time.Millisecond
)

func main() {
	log.Println("Starting Seat Killer...")

	// --- 1. Load Configs & Map ---
	userInfo, err := config.LoadUserInfo("user_info.yml")
	if err != nil {
		log.Fatalf("Failed to load user_info.yml: %v", err)
	}
	log.Println("User info loaded.")

	seatCfg, err := config.LoadSeatConfig("user_config.yml")
	if err != nil {
		log.Fatalf("Failed to load user_config.yml: %v", err)
	}
	log.Println("Seat config loaded.")

	if _, err = mapper.LoadSeatMap("seat_report.txt"); err != nil {
		log.Fatalf("Failed to load seat map: %v", err)
	}
	log.Println("Seat map loaded.")

	// --- 2. Validate Credentials ---
	log.Println("Validating user credentials...")
	if err := sso.ValidateCredentials(userInfo.SchoolID, userInfo.Password); err != nil {
		log.Fatalf("Credential validation failed: %v. Please check your user_info.yml.", err)
	}
	log.Println("User credentials are valid.")

	// --- 3. Determine Today's Booking Task ---
	weekdayMap := map[time.Weekday]string{
		time.Sunday: "周日", time.Monday: "周一", time.Tuesday: "周二",
		time.Wednesday: "周三", time.Thursday: "周四", time.Friday: "周五",
		time.Saturday: "周六",
	}
	todayWeekdayStr := weekdayMap[time.Now().Weekday()]

	dayConfig, ok := seatCfg.WeekConfig[todayWeekdayStr]
	if !ok || !dayConfig.Enable {
		log.Printf("Booking is not enabled for today (%s). Exiting.", todayWeekdayStr)
		return
	}
	log.Printf("Found booking task for today (%s): Run at %d:00 to book one of %d seat(s).",
		todayWeekdayStr, dayConfig.RunAtHour, len(dayConfig.Seats))

	// --- 4. Wait for the booking window ---
	now := time.Now()
	officialBookTime := time.Date(now.Year(), now.Month(), now.Day(), dayConfig.RunAtHour, 0, 0, 0, time.Local)
	preemptTime := officialBookTime.Add(-time.Duration(seatCfg.Global.PreemptSeconds) * time.Second)
	windowEndTime := officialBookTime.Add(windowMinutes * time.Minute)

	log.Printf("Official booking time is %s. Will start trying at %s.", officialBookTime.Format("15:04:05"), preemptTime.Format("15:04:05"))

	if now.Before(preemptTime) {
		time.Sleep(preemptTime.Sub(now))
	}

	if time.Now().After(windowEndTime) {
		log.Println("Booking window has already passed. Exiting.")
		return
	}

	log.Println("Booking window opened. Starting high-frequency requests...")

	// --- 5. High-Frequency Booking Loop ---
	client, _, err := sso.Login(userInfo.SchoolID, userInfo.Password)
	if err != nil {
		log.Fatalf("Initial login failed: %v", err)
	}
	loggedInUser, err := user.GetUserInfo(client)
	if err != nil {
		log.Fatalf("Initial user info fetch failed: %v", err)
	}
	log.Printf("Logged in as UID: %s. Ready to book.", loggedInUser.UID)

	bookingDay := time.Now().AddDate(0, 0, 2)
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	for t := range ticker.C {
		if t.After(windowEndTime) {
			log.Println("Booking window closed. Stopping attempts.")
			break
		}

		// In each tick, iterate through all seats according to priority.
		for _, seatNum := range dayConfig.Seats {
			seatID, err := mapper.GetSeatID(dayConfig.Name, seatNum)
			if err != nil {
				log.Printf("Cannot find seat '%s' in room '%s', skipping.", seatNum, dayConfig.Name)
				continue
			}

			bookTime := time.Date(bookingDay.Year(), bookingDay.Month(), bookingDay.Day(), dayConfig.BookStartHour, 0, 0, 0, time.Local)
			duration := time.Duration(dayConfig.Duration) * time.Hour

			log.Printf("Attempting to book room '%s', seat '%s' (%d)", dayConfig.Name, seatNum, seatID)
			result, err := booker.BookSeat(client, loggedInUser.UID, seatID, bookTime, duration)
			if err != nil {
				log.Printf("Booking attempt failed: %v", err)
				continue // Try next seat
			}

			log.Printf("Booking result: [%s] %s", result.CODE, result.MESSAGE)
			if result.CODE == "ok" {
				log.Println("BOOKING SUCCESSFUL! Exiting.")
				return // Exit main function on success
			}
			// If not "ok" (e.g., seat taken), the loop will continue to the next seat.
		}
	}

	log.Println("Seat Killer finished: all attempts failed within the window.")
}
