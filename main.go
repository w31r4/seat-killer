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
	// Start sending requests 1 minute before the official booking time.
	preemptMinutes = 1
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
		log.Fatalf("Failed to load user_info.yml: %v. Please make sure it exists and is correctly formatted.", err)
	}
	log.Println("User info loaded.")

	_, err = mapper.LoadSeatMap("seat_report.txt")
	if err != nil {
		log.Fatalf("Failed to load seat map: %v", err)
	}
	log.Println("Seat map loaded.")

	seatCfg, err := config.LoadSeatConfig("user_config.yml")
	if err != nil {
		log.Fatalf("Failed to load user_config.yml: %v", err)
	}
	log.Println("Seat config loaded.")

	// --- 2. Determine Today's Booking Task ---
	// The logic is: on the day of `run_at_hour`, we book a seat for two days later.
	// E.g., on Monday at 20:00, we book a seat for Wednesday.
	weekdayMap := map[time.Weekday]string{
		time.Sunday:    "周日",
		time.Monday:    "周一",
		time.Tuesday:   "周二",
		time.Wednesday: "周三",
		time.Thursday:  "周四",
		time.Friday:    "周五",
		time.Saturday:  "周六",
	}
	todayWeekdayStr := weekdayMap[time.Now().Weekday()]

	dayConfig, ok := seatCfg.WeekConfig[todayWeekdayStr]
	if !ok || !dayConfig.Enable {
		log.Printf("Booking is not enabled for today (%s). Exiting.", todayWeekdayStr)
		return
	}
	log.Printf("Found booking task for today (%s): Run at %d:00 to book a seat for 2 days later at %d:00.",
		todayWeekdayStr, dayConfig.RunAtHour, dayConfig.BookStartHour)

	// --- 3. Wait for the booking window ---
	now := time.Now()
	officialBookTime := time.Date(now.Year(), now.Month(), now.Day(), dayConfig.RunAtHour, 0, 0, 0, time.Local)
	preemptTime := officialBookTime.Add(-preemptMinutes * time.Minute)
	windowEndTime := officialBookTime.Add(windowMinutes * time.Minute)

	log.Printf("Official booking time is %s. Will start trying at %s.", officialBookTime.Format("15:04:05"), preemptTime.Format("15:04:05"))

	if now.Before(preemptTime) {
		waitDuration := preemptTime.Sub(now)
		log.Printf("Waiting for %v...", waitDuration)
		time.Sleep(waitDuration)
	}

	if now.After(windowEndTime) {
		log.Println("Booking window has already passed. Exiting.")
		return
	}

	log.Println("Booking window opened. Starting high-frequency requests...")

	// --- 4. High-Frequency Booking Loop ---
	client, _, err := sso.Login(userInfo.SchoolID, userInfo.Password)
	if err != nil {
		log.Fatalf("Initial login failed: %v", err)
	}
	loggedInUser, err := user.GetUserInfo(client)
	if err != nil {
		log.Fatalf("Initial user info fetch failed: %v", err)
	}
	log.Printf("Logged in as UID: %s. Ready to book.", loggedInUser.UID)

	seatID, err := mapper.GetSeatID(dayConfig.Name, dayConfig.Seat)
	if err != nil {
		log.Fatalf("Could not find seat '%s' in room '%s': %v", dayConfig.Seat, dayConfig.Name, err)
	}

	// Booking for two days in the future.
	bookingDay := time.Now().AddDate(0, 0, 2)
	bookTime := time.Date(bookingDay.Year(), bookingDay.Month(), bookingDay.Day(), dayConfig.BookStartHour, 0, 0, 0, time.Local)
	duration := time.Duration(dayConfig.Duration) * time.Hour

	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	for t := range ticker.C {
		if t.After(windowEndTime) {
			log.Println("Booking window closed. Stopping attempts.")
			break
		}

		log.Printf("Attempting to book seat %d for %s...", seatID, bookTime.Format("2006-01-02 15:04"))
		result, err := booker.BookSeat(client, loggedInUser.UID, seatID, bookTime, duration)
		if err != nil {
			log.Printf("Booking attempt failed: %v", err)
			continue // Try again on the next tick
		}

		log.Printf("Booking result: [%s] %s", result.CODE, result.MESSAGE)
		if result.CODE == "ok" {
			log.Println("BOOKING SUCCESSFUL! Exiting.")
			break // Exit loop on success
		}
	}

	log.Println("Seat Killer finished.")
}
