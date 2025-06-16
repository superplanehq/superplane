// login.go - Proper OAuth implementation
package cli

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const authSuccessTemplate = `
<!DOCTYPE html>
<html>
<head>
	<title>Authentication Successful</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; text-align: center; padding: 50px; }
		.success { color: #28a745; font-size: 24px; margin-bottom: 20px; }
		.message { color: #666; font-size: 16px; }
	</style>
</head>
<body>
	<div class="success">Authentication Successful!</div>
	<div class="message">You can now close this window and return to your terminal.</div>
	<script>
		setTimeout(function() {
			window.close();
		}, 10000);
	</script>
</body>
</html>
`

type AuthResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatar_url"`
	AccessToken string    `json:"access_token"`
	CreatedAt   time.Time `json:"created_at"`
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Superplane",
	Long:  `Login to Superplane using OAuth or development mode.`,
	Run: func(cmd *cobra.Command, args []string) {
		provider, _ := cmd.Flags().GetString("provider")
		devMode, _ := cmd.Flags().GetBool("dev")
		port, _ := cmd.Flags().GetInt("port")

		baseURL := GetAPIURL()

		if devMode || os.Getenv("APP_ENV") == "development" {
			fmt.Println("🔧 Development mode - using mock authentication")
			handleDevLogin(baseURL, provider)
			return
		}

		if provider == "" {
			provider = "github"
		}

		handleOAuthLogin(baseURL, provider, port)
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

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	CheckWithMessage(err, "Failed to parse auth response")

	// Store the token
	viper.Set(ConfigKeyAuthToken, authResp.AccessToken)
	err = viper.WriteConfig()
	CheckWithMessage(err, "Failed to save authentication token")

	fmt.Printf("✅ Successfully logged in as %s (%s)\n", authResp.Name, authResp.Email)
}

func handleOAuthLogin(baseURL, provider string, callbackPort int) {
	state := generateRandomState()

	callbackURL := fmt.Sprintf("http://localhost:%d/callback", callbackPort)
	tokenChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	server := startCallbackServer(callbackPort, state, tokenChan, errorChan)
	defer server.Shutdown(context.Background())

	authURL := fmt.Sprintf("%s/auth/%s?callback_url=%s&state=%s",
		baseURL, provider, url.QueryEscape(callbackURL), state)

	fmt.Printf("🌐 Opening browser for %s authentication...\n", provider)
	fmt.Printf("If browser doesn't open, visit: %s\n", authURL)
	fmt.Println("Waiting for authentication...")

	openBrowser(authURL)

	select {
	case token := <-tokenChan:
		// Store the token
		viper.Set(ConfigKeyAuthToken, token)
		err := viper.WriteConfig()
		CheckWithMessage(err, "Failed to save authentication token")

		fmt.Println("✅ Successfully authenticated!")
		fmt.Println("Token saved to config file.")

	case err := <-errorChan:
		Fail(fmt.Sprintf("Authentication failed: %v", err))

	case <-time.After(5 * time.Minute):
		Fail("Authentication timeout - please try again")
	}
}

func startCallbackServer(port int, expectedState string, tokenChan chan string, errorChan chan error) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			errorChan <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			error := r.URL.Query().Get("error")
			if error != "" {
				errorChan <- fmt.Errorf("authentication error: %s", error)
				http.Error(w, "Authentication failed", http.StatusBadRequest)
				return
			}

			errorChan <- fmt.Errorf("no token received")
			http.Error(w, "No token received", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(authSuccessTemplate))

		tokenChan <- token
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errorChan <- fmt.Errorf("callback server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return server
}

func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
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
	Short: "Logout from Superplane",
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

		baseURL := GetAPIURL()
		client := &http.Client{Timeout: 30 * time.Second}

		req, err := http.NewRequest("GET", baseURL+"/auth/me", nil)
		CheckWithMessage(err, "Failed to create request")

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		CheckWithMessage(err, "Failed to get user info")
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("Authentication token expired or invalid. Run 'superplane login' again.")
			os.Exit(1)
		}

		if resp.StatusCode != http.StatusOK {
			Fail(fmt.Sprintf("Failed to get user info: HTTP %d", resp.StatusCode))
		}

		var authResp AuthResponse
		err = json.NewDecoder(resp.Body).Decode(&authResp)
		CheckWithMessage(err, "Failed to parse user info")

		fmt.Printf("Logged in as: %s (%s)\n", authResp.Name, authResp.Email)
		fmt.Printf("User ID: %s\n", authResp.ID)
	},
}

func init() {
	RootCmd.AddCommand(loginCmd)
	RootCmd.AddCommand(logoutCmd)
	RootCmd.AddCommand(whoamiCmd)

	loginCmd.Flags().String("provider", "github", "OAuth provider (github, gitlab, bitbucket)")
	loginCmd.Flags().Bool("dev", false, "Use development mode authentication")
	loginCmd.Flags().Int("port", 8080, "Port for OAuth callback server")
}
