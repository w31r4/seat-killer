package user

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	userInfoURL = "https://hdu.huitu.zhishulib.com/Seat/Index/searchSeats?LAB_JSON=1"
)

// UserInfo matches the structure of the user data in the JSON response.
type UserInfo struct {
	UID       string `json:"uid"`
	UName     string `json:"uname"`
	UNickname string `json:"unickname"`
}

// ResponseData matches the top-level structure of the JSON response.
type ResponseData struct {
	DATA UserInfo `json:"DATA"`
}

// GetUserInfo fetches user information after a successful login.
func GetUserInfo(client *http.Client) (*UserInfo, error) {
	// The searchSeats endpoint requires a POST request, even for just getting user info.
	resp, err := client.Post(userInfoURL, "application/x-www-form-urlencoded;charset=UTF-8", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to user info url: %w", err)
	}
	defer resp.Body.Close()

	var responseData ResponseData
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to decode user info json: %w", err)
	}

	if responseData.DATA.UID == "" {
		return nil, fmt.Errorf("user uid not found in response")
	}

	return &responseData.DATA, nil
}
