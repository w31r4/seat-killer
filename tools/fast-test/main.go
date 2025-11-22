package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"seat-killer/booker"
	"seat-killer/config"
	"seat-killer/mapper"
	"seat-killer/sso"
	"seat-killer/user"
	"strconv"
	"time"
)

func main() {
	log.Println("--- Starting Fast-Test ---")

	// 1. Load All Configs
	userInfoPath := resolveFastTestPath("user_info.yml")
	seatCfgPath := resolveFastTestPath("user_config.yml")
	seatMapPath := resolveSeatReportPath()

	log.Printf("Loading user info from %s ...", userInfoPath)
	userInfo, err := config.LoadUserInfo(userInfoPath)
	if err != nil {
		log.Fatalf("Failed to load %s: %v", userInfoPath, err)
	}

	log.Printf("Loading seat config from %s ...", seatCfgPath)
	seatCfg, err := config.LoadSeatConfig(seatCfgPath)
	if err != nil {
		log.Fatalf("Failed to load %s: %v", seatCfgPath, err)
	}

	log.Printf("Loading seat map from %s ...", seatMapPath)
	if _, err = mapper.LoadSeatMap(seatMapPath); err != nil {
		log.Fatalf("Failed to load %s: %v", seatMapPath, err)
	}
	log.Println("All configurations loaded successfully.")

	// 2. Login
	log.Println("\n--- Testing Login ---")
	client, _, err := sso.Login(userInfo.SchoolID, userInfo.Password)
	if err != nil {
		log.Fatalf("Login test failed: %v", err)
	}
	log.Println("Login successful!")

	loggedInUser, err := user.GetUserInfo(client)
	if err != nil {
		log.Fatalf("Failed to fetch user info after login: %v", err)
	}
	log.Printf("Successfully fetched user info for UID: %s", loggedInUser.UID)

	// 3. Test Booking with an Invalid Seat from the dedicated test task
	log.Println("\n--- Testing Booking API with Invalid Seat ---")
	task, ok := seatCfg.WeekConfig["fast_test_task"]
	if !ok || !task.Enable || len(task.Seats) == 0 {
		log.Fatalf("The dedicated 'fast_test_task' is not configured correctly in user_config.yml.")
	}

	seatNum := task.Seats[0]
	log.Printf("Attempting to use an intentionally invalid seat: Room='%s', Seat='%s'", task.Name, seatNum)

	seatID, err := mapper.GetSeatID(task.Name, seatNum)
	if err != nil {
		log.Printf("As expected, seat '%s' in room '%s' is not in the local map: %v", seatNum, task.Name, err)
		if fakeID, parseErr := strconv.Atoi(seatNum); parseErr == nil {
			seatID = fakeID
		} else {
			seatID = 0 // fall back to an obviously invalid seat ID
		}
		log.Printf("Will send the request with a dummy seat ID (%d) to exercise the API.", seatID)
	} else {
		seatID += 1_000_000 // offset to avoid touching a real seat if it unexpectedly exists
		log.Printf("Seat unexpectedly exists locally (ID=%d). Offset to %d to keep the request invalid.", seatID-1_000_000, seatID)
	}

	bookTime := calculateBeginTime(time.Now())
	duration := time.Hour

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
		log.Printf("Booking API responded for begin time %s (duration %s):", bookTime.Format("2006-01-02 15:04:05"), duration)
		log.Printf("  CODE: %v", result.CODE)
		log.Printf("  MESSAGE: %s", result.MESSAGE)
		log.Printf("  IsSuccess(): %t", result.IsSuccess())
		if !result.IsSuccess() {
			log.Println("SUCCESS: The API correctly rejected the request for the invalid seat.")
		}
	}

	log.Println("\n--- Fast-Test Finished ---")
	fmt.Println("\nTip: A successful test means the program correctly identifies the invalid seat locally or the API rejects it.")
}

// calculateBeginTime picks a quick test start time:
// - If now is within 08:00-22:00, use now+1h (but never past 22:00).
// - Otherwise, use 08:00 of the next day.
func calculateBeginTime(now time.Time) time.Time {
	startWindow := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
	endWindow := time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, now.Location())

	if !now.Before(startWindow) && now.Before(endWindow) {
		candidate := now.Add(time.Hour)
		if candidate.After(endWindow) {
			return startWindow.Add(24 * time.Hour)
		}
		return candidate
	}

	return startWindow.Add(24 * time.Hour)
}

// resolveFastTestPath prefers files under tools/fast-test, but gracefully
// falls back to the current directory. This makes the tool runnable from
// either repo root or the fast-test folder.
func resolveFastTestPath(fileName string) string {
	candidates := []string{
		filepath.Join("tools", "fast-test", fileName),
		fileName,
		filepath.Join("..", fileName),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return fileName
}

// resolveSeatReportPath locates seat_report.txt relative to common launch points.
func resolveSeatReportPath() string {
	candidates := []string{
		"seat_report.txt",
		filepath.Join("..", "seat_report.txt"),
		filepath.Join("tools", "fast-test", "seat_report.txt"), // unlikely, but checked for completeness
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "seat_report.txt"
}
