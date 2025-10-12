package sso

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/hduLib/hdu/client"
	"github.com/hduLib/hdu/sso"
)

const loginURL = "https://sso.hdu.edu.cn/login?service=https:%2F%2Fhdu.huitu.zhishulib.com%2FUser%2FIndex%2FhduCASLogin%3Fforward%3D%252FSpace%252FCategory%252Fredirect%253Fcategory_id%253D591"

func Login(user, passwd string) (*http.Client, string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, "", err
	}
	// The hdu-go-lib client uses a global client, so we must replace it
	// with one that uses our jar.
	client.DefaultClient = &http.Client{
		Jar: jar,
	}

	// GenLoginReq will perform the login and all redirects.
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
		return nil, "", errors.New("PHPSESSID not found after login")
	}

	// Return the client (which has the jar) and the specific PHPSESSID.
	return client.DefaultClient.(*http.Client), phpSessID, nil
}
