package sso

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"seat-killer/retry"

	"github.com/hduLib/hdu/client"
	"github.com/hduLib/hdu/sso"
)

const (
	loginURL = "https://sso.hdu.edu.cn/login?service=https:%2F%2Fhdu.huitu.zhishulib.com%2FUser%2FIndex%2FhduCASLogin%3Fforward%3D%252FSpace%252FCategory%252Fredirect%253Fcategory_id%253D591"
	// A more realistic User-Agent to better mimic a real browser.
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0"
)

// customTransport injects a User-Agent header into each request.
type customTransport struct {
	http.RoundTripper
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)
	return t.RoundTripper.RoundTrip(req)
}

func Login(user, passwd string) (*http.Client, string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, "", err
	}

	// The hdu-go-lib client uses a global client. We must replace it with
	// one that uses our custom transport (for the User-Agent) and cookie jar.
	customClient := &http.Client{
		Jar: jar,
		Transport: &customTransport{
			RoundTripper: http.DefaultTransport,
		},
	}
	client.DefaultClient = customClient

	// GenLoginReq will perform the login and all redirects using our custom client.
	// The final session cookies will be stored in our jar.
	_, err = sso.GenLoginReq(loginURL, user, passwd)
	if err != nil {
		return nil, "", err
	}

	// After login, find the PHPSESSID from the jar.
	var phpSessID string
	targetURL, _ := url.Parse("https://hdu.huitu.zhishulib.com")
	for _, cookie := range jar.Cookies(targetURL) {
		if cookie.Name == "PHPSESSID" {
			phpSessID = cookie.Value
			break
		}
	}

	if phpSessID == "" {
		// This is a critical failure, likely due to incorrect credentials. Mark as unretryable.
		return nil, "", retry.WrapUnretryable(errors.New("PHPSESSID not found after login, please check your credentials"))
	}

	// Return the custom client (which has the jar) and the specific PHPSESSID.
	return customClient, phpSessID, nil
}

// ValidateCredentials attempts to log in to check if the user's credentials are valid.
// It does not retain the session cookie, making it a pure validation function.
func ValidateCredentials(user, passwd string) error {
	_, _, err := Login(user, passwd)
	return err
}
