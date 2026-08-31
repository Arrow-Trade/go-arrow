// auth.go
package arrow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// AuthResponse represents the structure of the authentication response from the API.
type AuthResponse struct {
	Status string `json:"status"` // API response status (e.g., "success" or "error").
	Data   struct {
		Name   string `json:"name"`   // User's name.
		Token  string `json:"token"`  // Authentication token.
		UserID string `json:"userId"` // Unique identifier for the user.
	} `json:"data"`
}

// GenerateChecksum creates a SHA256 hash of "appId:appSecret:request-token".
//
// This is used to securely authenticate API requests.
//
// Parameters:
//   - appID: The application ID.
//   - appSecret: The application secret key.
//   - requestToken: The temporary request token obtained from the login process.
//
// Returns:
//   - A SHA256 checksum string.
func GenerateChecksum(appID, appSecret, requestToken string) string {
	data := fmt.Sprintf("%s:%s:%s", appID, appSecret, requestToken)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Authenticate exchanges the request token for an access token.
//
// This function sends a POST request to authenticate the user and obtain an API token.
//
// Parameters:
//   - requestToken: The temporary token received after user login.
//
// Returns:
//   - A string containing the authentication token if successful.
//   - An error if authentication fails.
func (c *Client) Authenticate(requestToken string) (string, error) {
	checksum := GenerateChecksum(c.Config.AppID, c.Config.AppSecret, requestToken)

	payload := fmt.Sprintf(`{
		"checkSum": "%s",
		"checksum": "%s",
		"token": "%s",
		"appId": "%s"
	}`, checksum, checksum, requestToken, c.Config.AppID)

	responseBody, err := c.request("/auth/app/authenticate-token", "POST", []byte(payload))
	if err != nil {
		log.Error().Err(err).Msg("Failed to authenticate")
		return "", err
	}

	var authResponse AuthResponse
	if err := json.Unmarshal(responseBody, &authResponse); err != nil {
		log.Error().Err(err).Msg("Failed to parse authentication response")
		return "", err
	}

	if authResponse.Status != "success" {
		return "", fmt.Errorf("authentication failed: %s", authResponse.Status)
	}

	// Update client token after authentication
	c.Config.Token = authResponse.Data.Token

	c.debugf("Authentication successful", func(e *zerolog.Event) {
		e.Str("userID", authResponse.Data.UserID)
	})
	return authResponse.Data.Token, nil
}

// Login prompts the user to log in manually and enter the request token.
//
// This function prints a login URL and asks the user to enter the request token
// to complete the authentication process.
func (c *Client) Login() {
	loginURL := fmt.Sprintf("https://app.arrow.trade/app/login?appId=%s", c.Config.AppID)
	fmt.Println("Please visit the following URL to log in and retrieve your request token:")
	fmt.Println(loginURL)
	fmt.Println("After logging in, enter the request token below:")

	var requestToken string
	fmt.Print("Enter Request Token: ")
	fmt.Scanln(&requestToken)

	_, err := c.Authenticate(requestToken)
	if err != nil {
		log.Error().Err(err).Msg("Login authentication failed")
		return
	}

	fmt.Println("Authentication successful.")
}

// AutoLogin handles the entire authentication flow automatically using credentials.
//
// This function logs in a user programmatically by sending the credentials,
// performing 2FA verification using TOTP, extracting the request token, and
// exchanging it for an access token.
//
// Parameters:
//   - username: The user's registered ID or email.
//   - password: The user's password.
//   - totpSecret: The TOTP secret key used to generate 2FA codes.
//
// Returns:
//   - An error if authentication fails; otherwise, nil.
func (c *Client) AutoLogin(username, password, totpSecret string) error {
	loginURL := "https://api.arrow.trade/auth/app/login"

	// Step 1: Send Login Request
	payload := fmt.Sprintf(`{
		"userID": "%s",
		"password": "%s",
		"captchaValue": "",
		"captchaID": null,
		"appID": "%s",
		"isAppLogin": true
	}`, username, password, c.Config.AppID)

	resp, err := c.rawRequest(loginURL, "POST", []byte(payload))
	if err != nil {
		log.Error().Err(err).Msg("Login request failed")
		return err
	}

	var loginResp struct {
		Data struct {
			RequestID string `json:"requestId"` // Temporary request ID for 2FA validation.
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &loginResp); err != nil {
		log.Error().Err(err).Msg("Failed to parse login response")
		return err
	}

	// Step 2: Generate TOTP Code
	passcode, err := generateTOTP(totpSecret)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate TOTP code")
		return err
	}

	// Step 3: Validate 2FA
	totpPayload := fmt.Sprintf(`{
		"code": "%s",
		"requestId": "%s",
		"userID": "%s"
	}`, passcode, loginResp.Data.RequestID, username)

	resp, err = c.rawRequest("https://edge.arrow.trade/auth/validate-2fa", "POST", []byte(totpPayload))
	if err != nil {
		log.Error().Err(err).Msg("2FA validation failed")
		return err
	}

	var totpResp struct {
		Data struct {
			RedirectURL string `json:"redirectUrl"` // URL containing the request token.
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &totpResp); err != nil {
		log.Error().Err(err).Msg("Failed to parse 2FA response")
		return err
	}

	// Step 4: Extract Request Token from Redirect URL
	parsedURL, err := url.Parse(totpResp.Data.RedirectURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse redirect URL")
		return err
	}

	requestToken := parsedURL.Query().Get("request-token")

	// Step 5: Authenticate and Get Access Token
	_, err = c.Authenticate(requestToken)
	if err != nil {
		log.Error().Err(err).Msg("Authentication failed")
		return err
	}
	fmt.Println("AutoLogin successful.")
	return nil
}

// generateTOTP generates a TOTP (Time-based One-Time Password) code using a given secret.
//
// This function generates a 6-digit TOTP code that is valid for 30 seconds.
//
// Parameters:
//   - secret: The TOTP secret key.
//
// Returns:
//   - A string containing the generated TOTP code if successful.
//   - An error if TOTP generation fails.
func generateTOTP(secret string) (string, error) {
	return totp.GenerateCodeCustom(
		secret,
		time.Now(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    6,
			Algorithm: otp.AlgorithmSHA1,
		},
	)
}
