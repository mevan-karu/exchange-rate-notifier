package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

// Config struct to hold configuration
type Config struct {
	SMTPServer   string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	ToEmails     []string
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Get configuration from environment variables
	config := Config{
		SMTPServer:   getEnv("SMTP_SERVER", "smtp.sendgrid.net"),
		SMTPPort:     getEnv("SMTP_PORT", "465"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "mevan200@gmail.com"),
		ToEmails:     strings.Split(getEnv("TO_EMAILS", ""), ","),
	}

	// Validate configuration
	if config.SMTPUsername == "" || config.SMTPPassword == "" || config.FromEmail == "" || len(config.ToEmails) == 0 {
		log.Fatal("Missing required environment variables")
	}

	// Get the exchange rate
	exchangeRate, err := getSampathBankUSDRate()
	if err != nil {
		log.Fatalf("Error getting exchange rate: %v", err)
	}

	// Send the email
	err = sendEmail(config, exchangeRate)
	if err != nil {
		log.Fatalf("Error sending email: %v", err)
	}

	log.Printf("Email sent successfully with exchange rate: %s", exchangeRate)
}

func getSampathBankUSDRate() (string, error) {
	// Set up Chrome options
	opts := []selenium.ServiceOption{}
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}

	// Configure Chrome to run headless
	chromeCaps := chrome.Capabilities{
		Args: []string{
			"--headless",
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	}
	caps.AddChrome(chromeCaps)

	// Start the WebDriver service
	service, err := selenium.NewChromeDriverService("chromedriver", 9515, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to start ChromeDriver service: %v", err)
	}
	defer service.Stop()

	// Connect to the WebDriver
	driver, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", 9515))
	if err != nil {
		return "", fmt.Errorf("failed to connect to WebDriver: %v", err)
	}
	defer driver.Quit()

	// Navigate to the target page
	err = driver.Get("https://numbers.lk/trackers/exrates")
	if err != nil {
		return "", fmt.Errorf("failed to load page: %v", err)
	}

	// Wait for the page to load
	time.Sleep(5 * time.Second)

	// Get the page source
	source, err := driver.PageSource()
	if err != nil {
		return "", fmt.Errorf("failed to get page source: %v", err)
	}

	// Parse the HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(source))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Find Sampath Bank USD rate
	var rate string
	found := false

	// Find the table rows
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		// Check if this row contains Sampath Bank
		bankName := s.Find("td:first-child").Text()
		if strings.Contains(strings.ToLower(bankName), "sampath") {
			// Get the USD selling rate (typically in the third column)
			rate = s.Find("td:nth-child(3)").Text()
			rate = strings.TrimSpace(rate)
			found = true
		}
	})

	if !found {
		return "", fmt.Errorf("failed to find Sampath Bank USD rate")
	}

	return rate, nil
}

func sendEmail(config Config, exchangeRate string) error {
	// Compose the email
	subject := "Daily USD Exchange Rate Update"
	date := time.Now().Format("January 2, 2006")
	body := fmt.Sprintf("USD to LKR Exchange Rate at Sampath Bank as of %s: %s", date, exchangeRate)
	message := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=utf-8\r\n"+
		"\r\n"+
		"%s", strings.Join(config.ToEmails, ","), subject, body))

	// Set up authentication information
	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPServer)

	// Connect to the server, authenticate, set the sender and recipient, and send the email
	err := smtp.SendMail(
		config.SMTPServer+":"+config.SMTPPort,
		auth,
		config.FromEmail,
		config.ToEmails,
		message,
	)
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
