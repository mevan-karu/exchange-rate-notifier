package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Config struct to hold configuration
type Config struct {
	SendGridAPIKey string
	FromEmail      string
	ToEmails       []string
}

// ExchangeRateResponse represents the structure of the API response
type ExchangeRateResponse struct {
	Data []struct {
		Date            string      `json:"date"`
		Currency        string      `json:"currency"`
		Bank            string      `json:"bank"`
		BuyingCurrency  interface{} `json:"buying_currency"`
		SellingCurrency interface{} `json:"selling_currency"`
		CreatedTime     string      `json:"created_time"`
		EffectiveTime   string      `json:"effective_time"`
	} `json:"data"`
}

var config Config

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Get configuration from environment variables
	config = Config{
		SendGridAPIKey: getEnv("SENDGRID_API_KEY", ""),
		FromEmail:      getEnv("FROM_EMAIL", "mevan200@gmail.com"),
		ToEmails:       strings.Split(getEnv("TO_EMAILS", ""), ","),
	}

	// Validate configuration
	if config.SendGridAPIKey == "" || config.FromEmail == "" || len(config.ToEmails) == 0 {
		log.Fatalf("Required configuration(s) missing")
	}

	// Get the exchange rate
	exchangeRate, err := getSampathBankUSDRate()
	if err != nil {
		log.Fatalf("Error getting exchange rate: %v", err)
	}

	// Send the email
	for _, recipient := range config.ToEmails {
		err = sendEmail(recipient, exchangeRate)
		if err != nil {
			log.Printf("Error sending email to %s: %v", recipient, err)
		}
	}
	if err != nil {
		log.Fatalf("Error sending email: %v", err)
	}

	log.Printf("Email sent successfully with exchange rate: %s", exchangeRate)
}

func getSampathBankUSDRate() (string, error) {
	// Get current date in YYYY-MM-DD format
	currentDate := time.Now().Format("2006-01-02")

	// Construct the API URL with the current date
	apiUrl := fmt.Sprintf("https://cron.numbers.lk/api/exrates?currency=USD&date=%s&latest=true", currentDate)

	// Make HTTP GET request
	resp, err := http.Get(apiUrl)
	if err != nil {
		return "", fmt.Errorf("failed to make API request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the JSON response
	var exchangeRateResp ExchangeRateResponse
	if err := json.Unmarshal(body, &exchangeRateResp); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %v", err)
	}

	// Find Sampath Bank's USD buying rate
	for _, data := range exchangeRateResp.Data {
		if data.Bank == "SAMPATH" {
			// Handle different possible types (string or number)
			switch v := data.BuyingCurrency.(type) {
			case string:
				return v, nil
			case float64:
				return fmt.Sprintf("%.4f", v), nil
			default:
				return fmt.Sprintf("%v", v), nil
			}
		}
	}

	return "", fmt.Errorf("sampath bank USD exchange rate not found in the response")
}

func sendEmail(recipient string, exchangeRate string) error {
	from := mail.NewEmail("Exchange Rate Notifier", config.FromEmail)
	subject := fmt.Sprintf("Sampath Bank USD Exchange Rate: %s", exchangeRate)
	to := mail.NewEmail("Recipient", recipient)
	plainTextContent := fmt.Sprintf("Sampath Bank USD exchange rate is %s", exchangeRate)
	htmlContent := fmt.Sprintf("<strong>Sampath Bank USD exchange rate is %s</strong>", exchangeRate)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(config.SendGridAPIKey)
	_, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	return nil
}

// Helper function to get environment variable with default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
