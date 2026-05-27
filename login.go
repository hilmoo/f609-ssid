package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func loginToRouter(config Config) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	client := &http.Client{
		Jar: jar,
	}

	req, err := http.NewRequest("GET", config.RouterURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to router: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	htmlContent := string(bodyBytes)

	checkTokenRe := regexp.MustCompile(`"Frm_Loginchecktoken",\s*"([^"]+)"`)
	loginTokenRe := regexp.MustCompile(`"Frm_Logintoken",\s*"([^"]+)"`)

	var csrfToken string
	if matches := checkTokenRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
		csrfToken = matches[1]
	} else {
		fmt.Println("Warning: Could not extract CSRF check token.")
	}

	var loginToken string
	if matches := loginTokenRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
		loginToken = matches[1]
	} else {
		fmt.Println("Warning: Could not extract login token.")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomNum := r.Intn(90000000) + 10000000
	randomNumStr := fmt.Sprintf("%d", randomNum)

	hashInput := config.Password + randomNumStr
	hash := sha256.Sum256([]byte(hashInput))
	passwordHash := hex.EncodeToString(hash[:])

	formData := url.Values{}
	formData.Set("action", "login")
	formData.Set("Username", config.Username)
	formData.Set("Password", passwordHash)
	formData.Set("UserRandomNum", randomNumStr)
	formData.Set("Frm_Logintoken", loginToken)
	formData.Set("Frm_Loginchecktoken", csrfToken)

	postReq, err := http.NewRequest("POST", config.RouterURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %v", err)
	}

	respPost, err := client.Do(postReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send login request: %v", err)
	}
	defer respPost.Body.Close()

	cookieFound := false

	parsedURL, _ := url.Parse(config.RouterURL)
	cookies := jar.Cookies(parsedURL)

	for _, cookie := range cookies {
		if cookie.Name == "SID" {
			cookieFound = true
		}
	}

	if cookieFound {
		fmt.Println("Login successful, cookie acquired!")
		return client, nil
	}

	return nil, fmt.Errorf("login failed: session cookie not found")
}
