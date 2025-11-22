package booker

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	bookURL = "https://hdu.huitu.zhishulib.com/Seat/Index/bookSeats?LAB_JSON=1"
)

// BookResponseData matches the structure of the booking response.
type BookResponseData struct {
	CODE    interface{} `json:"CODE"`
	MESSAGE string      `json:"MESSAGE"`
	DATA    struct {
		BookingID string `json:"bookingId"`
	} `json:"DATA"`
}

// IsSuccess checks if the booking response indicates success.
// It handles cases where CODE might be a string ("ok") or other types.
func (r *BookResponseData) IsSuccess() bool {
	if codeStr, ok := r.CODE.(string); ok {
		return codeStr == "ok"
	}
	return false
}

// getApiToken generates the required api-token header value.
// Based on reverse-engineering of the library's web page, it's an MD5 hash.
func getApiToken(apiTime string) string {
	hash := md5.Sum([]byte("" + apiTime))
	return hex.EncodeToString(hash[:])
}

// BookSeat attempts to book a specific seat using the parameters from the request DTO.
func BookSeat(req *BookingRequest) (*BookResponseData, error) {
	// The python script calculates beginTime from the beginning of the current day.
	// The curl command uses a direct timestamp. Let's follow the curl command.
	beginTimestamp := req.BeginTime.Unix()
	apiTimestamp := time.Now().Unix()

	formData := url.Values{}
	formData.Set("beginTime", strconv.FormatInt(beginTimestamp, 10))
	formData.Set("duration", fmt.Sprintf("%.0f", req.Duration.Seconds()))
	formData.Set("seats[0]", strconv.Itoa(req.SeatID))
	formData.Set("seatBookers[0]", req.UserID)
	formData.Set("is_recommend", "1")
	formData.Set("api_time", strconv.FormatInt(apiTimestamp, 10))

	httpReq, err := http.NewRequest("POST", bookURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	// Set headers based on the curl command
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	httpReq.Header.Set("Accept", "application/json, text/plain, */*")
	httpReq.Header.Set("api-token", getApiToken(strconv.FormatInt(apiTimestamp, 10)))
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0")
	httpReq.Header.Set("Referer", "https://hdu.huitu.zhishulib.com/")

	resp, err := req.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var bookData BookResponseData
	if err := json.NewDecoder(resp.Body).Decode(&bookData); err != nil {
		return nil, fmt.Errorf("failed to decode book response: %w", err)
	}

	return &bookData, nil
}
