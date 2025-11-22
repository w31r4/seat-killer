package main

import (
	"log"
	"net/http"
	"seat-killer/booker"
	"seat-killer/config"
	"seat-killer/mapper"
	"seat-killer/sso"
	"seat-killer/user"
	"time"
)

func main() {
	log.Println("--- Starting Fast-Test ---")

	// 1. Load All Configs
	log.Println("Loading configurations...")
	userInfo, err := config.LoadUserInfo("user_info.yml")
	if err != nil {
		log.Fatalf("Failed to load user_info.yml: %v", err)
	}

	seatCfg, err := config.LoadSeatConfig("user_config.yml")
	if err != nil {
		log.Fatalf("Failed to load user_config.yml: %v", err)
	}

	if _, err = mapper.LoadSeatMap("seat_report.txt"); err != nil {
		log.Fatalf("Failed to load seat_map.txt: %v", err)
	}
	log.Println("All configurations loaded successfully.")

	// 2. Login
	log.Println("\n--- Testing Login ---")
	var client *http.Client
	var loggedInUser *user.UserInfo
	loginFunc := func() (err error) {
		client, _, err = sso.Login(userInfo.SchoolID, userInfo.Password)
		if err != nil {
			return
		}
		loggedInUser, err = user.GetUserInfo(client)
		return
	}

	if err := loginFunc(); err != nil {
		log.Fatalf("Login or user info fetch failed: %v", err)
	}
	log.Printf("Login successful! Fetched user info for UID: %s", loggedInUser.UID)

	// 3. Find First Enabled Task and Test Booking
	log.Println("\n--- Finding Task and Testing Booking ---")
	var taskFound bool
	for day, dayConfig := range seatCfg.WeekConfig {
		if dayConfig.Enable && len(dayConfig.Seats) > 0 {
			log.Printf("Found enabled task for '%s'. Preparing to test with the first seat.", day)
			taskFound = true

			seatNum := dayConfig.Seats[0]
			seatID, err := mapper.GetSeatID(dayConfig.Name, seatNum)
			if err != nil {
				log.Printf("Could not find seat ID for room '%s', seat '%s'. Skipping.", dayConfig.Name, seatNum)
				continue
			}

			// Use a fixed future time for testing, e.g., two days from now, at the configured start hour.
			bookTime := time.Date(
				time.Now().Year(), time.Now().Month(), time.Now().Day()+2,
				dayConfig.BookStartHour, 0, 0, 0, time.Local,
			)
			duration := time.Duration(dayConfig.Duration) * time.Hour

			log.Printf("Attempting to book seat: Room='%s', Seat='%s' (%d), Time='%s', Duration='%d hours'",
				dayConfig.Name, seatNum, seatID, bookTime.Format("2006-01-02 15:04"), dayConfig.Duration)

			bookReq := &booker.BookingRequest{
				Client:    client,
				UserID:    loggedInUser.UID,
				SeatID:    seatID,
				BeginTime: bookTime,
				Duration:  duration,
			}

			result, err := booker.BookSeat(bookReq)
			if err != nil {
				log.Printf("Booking request failed with an error: %v", err)
			}

			if result != nil {
				log.Println("Received a response from the booking API:")
				log.Printf("  CODE: %v", result.CODE)
				log.Printf("  MESSAGE: %s", result.MESSAGE)
				log.Printf("  IsSuccess(): %t", result.IsSuccess())
			}
			break // Test only the first found task
		}
	}

	if !taskFound {
		log.Println("No enabled booking tasks found in user_config.yml. Test cannot proceed.")
	}

	log.Println("\n--- Fast-Test Finished ---")
}
