package main

import (
	"fmt"
	"log"
	"seat-killer/booker"
	"seat-killer/config"
	"seat-killer/sso"
	"seat-killer/user"
	"time"
)

func main() {
	log.Println("--- Starting Fast-Test ---")

	// 1. Load User Info
	log.Println("Loading user credentials from user_info.yml...")
	userInfo, err := config.LoadUserInfo("user_info.yml")
	if err != nil {
		log.Fatalf("Failed to load user_info.yml: %v", err)
	}
	log.Printf("Loaded credentials for SchoolID: %s", userInfo.SchoolID)

	// 2. Test Login
	log.Println("\n--- Testing Login ---")
	log.Println("Attempting to log in...")
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

	// 3. Test Booking an Invalid Seat
	log.Println("\n--- Testing Booking API with Invalid Seat ---")
	invalidSeatID := 0 // Using 0 as it's an invalid seat ID
	log.Printf("Attempting to book an invalid seat (ID: %d) to check API response...", invalidSeatID)

	// Use a fixed future time for testing
	bookTime := time.Now().Add(48 * time.Hour)
	duration := 1 * time.Hour

	result, err := booker.BookSeat(client, loggedInUser.UID, invalidSeatID, bookTime, duration)
	if err != nil {
		log.Printf("Booking request failed with an error: %v", err)
		log.Println("This might be expected if the API gateway rejects the request before it hits the booking logic.")
	}

	if result != nil {
		log.Println("Received a response from the booking API:")
		log.Printf("  CODE: %v (Type: %T)", result.CODE, result.CODE)
		log.Printf("  MESSAGE: %s", result.MESSAGE)
		log.Printf("  IsSuccess(): %t", result.IsSuccess())
	}

	log.Println("\n--- Fast-Test Finished ---")
	fmt.Println("\nTip: If login fails, check your credentials in user_info.yml.")
	fmt.Println("The booking test helps understand how the server responds to invalid requests. A non-nil response is a good sign of successful communication.")
}
