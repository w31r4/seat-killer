package main

import (
	"log"
	"net/http"
	"time"

	"seat-killer/booker"
	"seat-killer/config"
	"seat-killer/mapper"
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
	userInfo, err := config.LoadUserInfo("user_info.yml")
	if err != nil {
		log.Fatalf("Failed to load user_info.yml: %v", err)
	}
	seatCfg, err := config.LoadSeatConfig("user_config.yml")
	if err != nil {
		log.Fatalf("Failed to load user_config.yml: %v", err)
	}
	if _, err = mapper.LoadSeatMap("seat_report.txt"); err != nil {
		log.Fatalf("Failed to load seat map: %v", err)
	}
	log.Println("Configs and seat map loaded.")

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
	if !ok || !dayConfig.Enable || len(dayConfig.Seats) == 0 {
		log.Printf("Booking is not enabled for today (%s) or no seats configured. Exiting.", todayWeekdayStr)
		return
	}
	log.Printf("Found booking task for today (%s): Run at %d:00 to book one of %d seat(s).",
		todayWeekdayStr, dayConfig.RunAtHour, len(dayConfig.Seats))

	// --- 4. Define Time Windows ---
	now := time.Now()
	officialBookTime := time.Date(now.Year(), now.Month(), now.Day(), dayConfig.RunAtHour, 0, 0, 0, time.Local)
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
	client, _, err := sso.Login(userInfo.SchoolID, userInfo.Password)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	loggedInUser, err := user.GetUserInfo(client)
	if err != nil {
		log.Fatalf("User info fetch failed: %v", err)
	}
	log.Printf("Logged in as UID: %s. Starting high-frequency requests...", loggedInUser.UID)

	// --- 7. Execute Phased Booking ---
	if executeBookingPhase(client, loggedInUser, &dayConfig, preemptTime, officialBookTime, true) {
		log.Println("BOOKING SUCCESSFUL in Attack Phase!")
		return
	}
	if executeBookingPhase(client, loggedInUser, &dayConfig, officialBookTime, fallbackEndTime, false) {
		log.Println("BOOKING SUCCESSFUL in Fallback Phase!")
		return
	}

	log.Println("Seat Killer finished: all attempts failed within all windows.")
}

// executeBookingPhase runs the booking loop for a specific time window and seat strategy.
// Returns true if booking was successful.
func executeBookingPhase(client *http.Client, user *user.UserInfo, dayCfg *config.DayConfig, start, end time.Time, primaryOnly bool) bool {
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	seatsToTry := dayCfg.Seats
	if primaryOnly {
		seatsToTry = []string{dayCfg.Seats[0]}
		log.Printf("--- Entering Attack Phase: Focusing on primary seat %s ---", seatsToTry[0])
	} else {
		log.Printf("--- Entering Fallback Phase: Trying all %d seats ---", len(seatsToTry))
	}

	bookingDay := time.Now().AddDate(0, 0, 2)

	for t := range ticker.C {
		if t.Before(start) {
			continue
		}
		if t.After(end) {
			return false
		}

		for _, seatNum := range seatsToTry {
			seatID, err := mapper.GetSeatID(dayCfg.Name, seatNum)
			if err != nil {
				log.Printf("Cannot find seat '%s' in room '%s', skipping.", seatNum, dayCfg.Name)
				continue
			}

			bookTime := time.Date(bookingDay.Year(), bookingDay.Month(), bookingDay.Day(), dayCfg.BookStartHour, 0, 0, 0, time.Local)
			duration := time.Duration(dayCfg.Duration) * time.Hour

			log.Printf("Attempting to book room '%s', seat '%s' (%d)", dayCfg.Name, seatNum, seatID)
			result, err := booker.BookSeat(client, user.UID, seatID, bookTime, duration)
			if err != nil {
				log.Printf("Booking attempt failed: %v", err)
				continue
			}

			log.Printf("Booking result: [%v] %s", result.CODE, result.MESSAGE)
			if result.IsSuccess() {
				return true
			}
		}
	}
	return false // Should not be reached if end time is handled correctly
}
