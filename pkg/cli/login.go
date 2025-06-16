package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GitHubClientID set during build time
var GitHubClientID string

const (
	GitHubDeviceCodeURL = "https://github.com/login/device/code"
	GitHubTokenURL      = "https://github.com/login/oauth/access_token"
	GitHubUserURL       = "https://api.github.com/user"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type TokenExchangeRequest struct {
	GitHubToken string `json:"github_token"`
}

type DevAuthResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatar_url"`
	AccessToken string    `json:"access_token"`
	CreatedAt   time.Time `json:"created_at"`
}

const mockedToken = "dev_token"

type TokenExchangeResponse struct {
	AccessToken string `json:"access_token"`
	User        struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
}

type ServerUserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub",
	Long:  `Login to GitHub using device flow authentication.`,
	Run: func(cmd *cobra.Command, args []string) {
		devMode, _ := cmd.Flags().GetBool("dev")
		provider, _ := cmd.Flags().GetString("provider")

		if devMode || os.Getenv("APP_ENV") == "development" {
			fmt.Printf("🔧 Development mode - using mock authentication with %s\n", provider)
			handleDevLogin(GetAPIURL(), provider)
			return
		}

		switch provider {
		case "github":
			handleGitHubDeviceFlow()
		default:
			Fail(fmt.Sprintf("Unsupported provider: %s. Currently supported: github", provider))
		}
	},
}

func handleDevLogin(baseURL, provider string) {
	fmt.Printf("🔧 Authenticating with %s in development mode...\n", provider)

	client := &http.Client{Timeout: 30 * time.Second}
	authURL := fmt.Sprintf("%s/auth/%s", baseURL, provider)

	req, err := http.NewRequest("GET", authURL, nil)
	CheckWithMessage(err, "Failed to create auth request")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	CheckWithMessage(err, "Failed to authenticate")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Fail(fmt.Sprintf("Authentication failed with status: %d", resp.StatusCode))
	}

	var authResp DevAuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	CheckWithMessage(err, "Failed to parse auth response")

	// Store the token
	viper.Set(ConfigKeyAuthToken, authResp.AccessToken)
	err = viper.WriteConfig()
	CheckWithMessage(err, "Failed to save authentication token")

	fmt.Printf("✅ Successfully logged in as %s (%s)\n", authResp.Name, authResp.Email)
}

func handleGitHubDeviceFlow() {
	fmt.Println("🔐 Starting GitHub authentication...")

	deviceResp, err := requestDeviceCode()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get device code: %v", err))
	}

	fmt.Printf("\n📱 Please visit: %s\n", deviceResp.VerificationURI)
	fmt.Printf("🔑 Enter this code: %s\n\n", deviceResp.UserCode)
	fmt.Println("Opening browser automatically...")

	openBrowser(deviceResp.VerificationURI)

	fmt.Println("⏳ Waiting for you to authorize the application...")

	githubToken, err := pollForToken(deviceResp.DeviceCode, deviceResp.Interval, deviceResp.ExpiresIn)
	if err != nil {
		Fail(fmt.Sprintf("GitHub authentication failed: %v", err))
	}

	fmt.Println("\n✅ GitHub authentication successful!")
	fmt.Println("🔄 Exchanging token with server...")

	appToken, user, err := exchangeGitHubToken(githubToken)
	if err != nil {
		Fail(fmt.Sprintf("Token exchange failed: %v", err))
	}

	viper.Set(ConfigKeyAuthToken, appToken)
	err = viper.WriteConfig()
	CheckWithMessage(err, "Failed to save authentication token")

	fmt.Printf("✅ Successfully authenticated as %s (%s)\n", user.User.Name, user.User.Email)
}

func getClientID() string {
	if clientID := os.Getenv("GITHUB_CLIENT_ID"); clientID != "" {
		return clientID
	}

	if GitHubClientID != "" {
		return GitHubClientID
	}

	return "invalid-client-id"
}

func getTokenExchangeURL() string {
	baseURL := GetAPIURL()
	return baseURL + "/auth/token/exchange"
}

func requestDeviceCode() (*DeviceCodeResponse, error) {
	clientID := getClientID()

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", "read:user user:email")

	req, err := http.NewRequest("POST", GitHubDeviceCodeURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "superplane-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer := bytes.Buffer{}
	buffer.ReadFrom(resp.Body)
	body := buffer.String()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d, body: %s", resp.StatusCode, body)
	}

	var deviceResp DeviceCodeResponse
	err = json.Unmarshal([]byte(body), &deviceResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v, body: %s", err, body)
	}

	return &deviceResp, nil
}

func pollForToken(deviceCode string, interval, expiresIn int) (string, error) {
	timeout := time.After(time.Duration(expiresIn) * time.Second)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("authentication timeout")
		case <-ticker.C:
			token, err := checkForToken(deviceCode)
			if err != nil {
				if err.Error() == "authorization_pending" {
					fmt.Print(".")
					continue
				}
				if err.Error() == "slow_down" {
					fmt.Print(".")
					ticker.Stop()
					ticker = time.NewTicker(time.Duration(interval+5) * time.Second)
					continue
				}
				return "", err
			}
			return token, nil
		}
	}
}

func checkForToken(deviceCode string) (string, error) {
	clientID := getClientID()

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequest("POST", GitHubTokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "superplane-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", err
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("GitHub API error: %s", tokenResp.Error)
	}

	return tokenResp.AccessToken, nil
}

func exchangeGitHubToken(githubToken string) (string, *TokenExchangeResponse, error) {
	reqBody := TokenExchangeRequest{
		GitHubToken: githubToken,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("POST", getTokenExchangeURL(), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "superplane-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenExchangeResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", nil, err
	}

	return tokenResp.AccessToken, &tokenResp, nil
}

// Updated to use your server's /auth/me endpoint instead of GitHub's API
func getUserInfo(token string) (*GitHubUser, error) {
	baseURL := GetAPIURL()
	req, err := http.NewRequest("GET", baseURL+"/auth/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "superplane-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Server returned status %d", resp.StatusCode)
	}

	var user ServerUserResponse
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &GitHubUser{
		Login:     user.Email,
		Name:      user.Name,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
	}, nil
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Println("Please open the URL manually in your browser.")
	}
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from GitHub",
	Long:  `Remove stored authentication token.`,
	Run: func(cmd *cobra.Command, args []string) {
		viper.Set(ConfigKeyAuthToken, "")
		err := viper.WriteConfig()
		CheckWithMessage(err, "Failed to clear authentication token")

		fmt.Println("✅ Successfully logged out")
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user information",
	Long:  `Display information about the currently authenticated user.`,
	Run: func(cmd *cobra.Command, args []string) {
		token := GetAuthToken()
		if token == "" {
			fmt.Println("Not authenticated. Run 'superplane login' first.")
			os.Exit(1)
		}

		if token == mockedToken {
			fmt.Println("Logged in as: Dev User (dev@superplane.local)")
			fmt.Println("User ID: dev-user-123")
			fmt.Println("Email: dev@superplane.local")
			fmt.Println("Mode: Development")
			return
		}

		user, err := getUserInfo(token)
		if err != nil {
			fmt.Println("Authentication token expired or invalid. Run 'superplane login' again.")
			os.Exit(1)
		}

		fmt.Printf("Logged in as: %s (%s)\n", user.Name, user.Login)
		if user.Email != "" {
			fmt.Printf("Email: %s\n", user.Email)
		}
	},
}

func init() {
	RootCmd.AddCommand(loginCmd)
	RootCmd.AddCommand(logoutCmd)
	RootCmd.AddCommand(whoamiCmd)

	loginCmd.Flags().Bool("dev", false, "Use development mode authentication")
	loginCmd.Flags().String("provider", "github", "OAuth provider to use (github)")
}
