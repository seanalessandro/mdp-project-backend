package services

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"gopkg.in/gomail.v2"
)

// EmailService struct to hold email configuration
// Similar to @Service in Spring Boot or Service class in Laravel
type EmailService struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromName  string
	FromEmail string
}

// NewEmailService creates a new email service instance
// Similar to @Autowired constructor in Spring Boot
func NewEmailService() *EmailService {
	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		port = 587 // default port
	}

	return &EmailService{
		Host:      os.Getenv("SMTP_HOST"),
		Port:      port,
		Username:  os.Getenv("SMTP_USERNAME"),
		Password:  os.Getenv("SMTP_PASSWORD"),
		FromName:  os.Getenv("SMTP_FROM_NAME"),
		FromEmail: os.Getenv("SMTP_FROM_EMAIL"),
	}
}

// GenerateRandomPassword generates a random password
// Similar to utility methods in Spring Boot or Laravel
func GenerateRandomPassword(length int) string {
	// Character set for password generation
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)

	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			log.Printf("Error generating random number: %v", err)
			// Fallback to simple random
			password[i] = charset[i%len(charset)]
		} else {
			password[i] = charset[num.Int64()]
		}
	}

	return string(password)
}

// SendWelcomeEmail sends a welcome email with password to new user
// Similar to @Async methods in Spring Boot or Mail::send() in Laravel
func (e *EmailService) SendWelcomeEmail(toEmail, fullName, username, password string) error {
	// Create new message
	m := gomail.NewMessage()

	// Set email headers
	m.SetHeader("From", m.FormatAddress(e.FromEmail, e.FromName))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Welcome to MDP System - Your Account Details")

	// Create HTML email body
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to MDP System</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
            <h1 style="color: #007bff; margin-bottom: 10px;">Welcome to MDP System</h1>
            <p style="font-size: 16px; margin-bottom: 0;">Your account has been successfully created!</p>
        </div>
        
        <div style="background-color: #ffffff; padding: 20px; border: 1px solid #dee2e6; border-radius: 8px;">
            <h2 style="color: #333; margin-bottom: 15px;">Account Details</h2>
            
            <p><strong>Full Name:</strong> %s</p>
            <p><strong>Username:</strong> %s</p>
            <p><strong>Email:</strong> %s</p>
            
            <div style="background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 5px; margin: 20px 0;">
                <h3 style="color: #856404; margin-bottom: 10px;">🔐 Your Temporary Password</h3>
                <p style="font-family: 'Courier New', monospace; font-size: 18px; background-color: #fff; padding: 10px; border: 1px solid #ccc; border-radius: 4px; text-align: center; color: #333;">
                    <strong>%s</strong>
                </p>
                <p style="color: #856404; font-size: 14px; margin-bottom: 0;">
                    ⚠️ Please change this password after your first login for security reasons.
                </p>
            </div>
            
            <div style="margin-top: 20px; padding-top: 20px; border-top: 1px solid #dee2e6;">
                <h3 style="color: #333;">Next Steps:</h3>
                <ol style="color: #666;">
                    <li>Login using your username and temporary password</li>
                    <li>Change your password immediately</li>
                    <li>Complete your profile if needed</li>
                </ol>
            </div>
        </div>
        
        <div style="text-align: center; margin-top: 20px; color: #666; font-size: 14px;">
            <p>If you have any questions, please contact your system administrator.</p>
            <p>&copy; 2025 MDP System. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, fullName, username, toEmail, password)

	// Set email body
	m.SetBody("text/html", htmlBody)

	// Create text alternative for email clients that don't support HTML
	textBody := fmt.Sprintf(`
Welcome to MDP System!

Your account has been successfully created.

Account Details:
- Full Name: %s
- Username: %s
- Email: %s
- Temporary Password: %s

Important: Please change this password after your first login for security reasons.

Next Steps:
1. Login using your username and temporary password
2. Change your password immediately  
3. Complete your profile if needed

If you have any questions, please contact your system administrator.

© 2025 MDP System. All rights reserved.
`, fullName, username, toEmail, password)

	m.AddAlternative("text/plain", textBody)

	// Create SMTP dialer
	d := gomail.NewDialer(e.Host, e.Port, e.Username, e.Password)

	// Send email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("Welcome email sent successfully to %s", toEmail)
	return nil
}

// SendPasswordResetEmail sends password reset email
// Additional method for password reset functionality
func (e *EmailService) SendPasswordResetEmail(toEmail, fullName, newPassword string) error {
	m := gomail.NewMessage()

	m.SetHeader("From", m.FormatAddress(e.FromEmail, e.FromName))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "MDP System - Password Reset")

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset - MDP System</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
            <h1 style="color: #dc3545; margin-bottom: 10px;">Password Reset</h1>
            <p style="font-size: 16px; margin-bottom: 0;">Your password has been reset by system administrator.</p>
        </div>
        
        <div style="background-color: #ffffff; padding: 20px; border: 1px solid #dee2e6; border-radius: 8px;">
            <h2 style="color: #333; margin-bottom: 15px;">Hello, %s</h2>
            
            <div style="background-color: #f8d7da; border: 1px solid #f5c6cb; padding: 15px; border-radius: 5px; margin: 20px 0;">
                <h3 style="color: #721c24; margin-bottom: 10px;">🔐 Your New Password</h3>
                <p style="font-family: 'Courier New', monospace; font-size: 18px; background-color: #fff; padding: 10px; border: 1px solid #ccc; border-radius: 4px; text-align: center; color: #333;">
                    <strong>%s</strong>
                </p>
                <p style="color: #721c24; font-size: 14px; margin-bottom: 0;">
                    ⚠️ Please change this password immediately after login for security reasons.
                </p>
            </div>
        </div>
        
        <div style="text-align: center; margin-top: 20px; color: #666; font-size: 14px;">
            <p>If you did not request this password reset, please contact your system administrator immediately.</p>
            <p>&copy; 2025 MDP System. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, fullName, newPassword)

	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(e.Host, e.Port, e.Username, e.Password)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send password reset email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send password reset email: %v", err)
	}

	log.Printf("Password reset email sent successfully to %s", toEmail)
	return nil
}

// SendDocumentStatusEmail sends notification when document status changes
// Similar to notification emails in enterprise document management systems
func (e *EmailService) SendDocumentStatusEmail(toEmail, authorName, documentTitle, status, reviewerName string) error {
	m := gomail.NewMessage()

	m.SetHeader("From", m.FormatAddress(e.FromEmail, e.FromName))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", fmt.Sprintf("Document Status Update: %s", documentTitle))

	// Determine status color and icon based on status
	var statusColor, statusIcon, statusMessage string
	switch status {
	case "In Review", "in_review", "review":
		statusColor = "#ffc107"
		statusIcon = "👁️"
		statusMessage = "Your document has been submitted for review and is now being evaluated by our review team."
	case "Approved", "approved":
		statusColor = "#28a745"
		statusIcon = "✅"
		statusMessage = "Congratulations! Your document has been approved and is now published."
	case "Rejected", "rejected":
		statusColor = "#dc3545"
		statusIcon = "❌"
		statusMessage = "Your document requires revisions. Please check the feedback and resubmit."
	case "Draft", "draft":
		statusColor = "#6c757d"
		statusIcon = "📝"
		statusMessage = "Your document has been saved as draft. You can continue editing when ready."
	default:
		statusColor = "#17a2b8"
		statusIcon = "📄"
		statusMessage = fmt.Sprintf("Your document status has been updated to: %s", status)
	}

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Document Status Update - MDP System</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
            <h1 style="color: #007bff; margin-bottom: 10px;">Document Status Update</h1>
            <p style="font-size: 16px; margin-bottom: 0;">Your document status has been updated in the MDP System.</p>
        </div>
        
        <div style="background-color: #ffffff; padding: 20px; border: 1px solid #dee2e6; border-radius: 8px;">
            <h2 style="color: #333; margin-bottom: 15px;">Hello, %s</h2>
            
            <div style="background-color: #ffffff; border: 2px solid %s; padding: 20px; border-radius: 8px; margin: 20px 0;">
                <div style="text-align: center; margin-bottom: 15px;">
                    <span style="font-size: 48px;">%s</span>
                </div>
                <h3 style="color: %s; text-align: center; margin-bottom: 15px; font-size: 24px;">
                    Status: %s
                </h3>
                <p style="text-align: center; font-size: 16px; color: #666; margin-bottom: 0;">
                    %s
                </p>
            </div>
            
            <div style="background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0;">
                <h3 style="color: #333; margin-bottom: 10px;">📄 Document Details</h3>
                <p><strong>Document Title:</strong> %s</p>
                <p><strong>Author:</strong> %s</p>
                <p><strong>Status:</strong> %s</p>
                %s
                <p><strong>Updated:</strong> %s</p>
            </div>
            
            <div style="margin-top: 20px; padding-top: 20px; border-top: 1px solid #dee2e6;">
                <h3 style="color: #333;">Next Steps:</h3>
                <ul style="color: #666;">
                    %s
                </ul>
            </div>
        </div>
        
        <div style="text-align: center; margin-top: 20px; color: #666; font-size: 14px;">
            <p>You can view your document in the MDP System dashboard.</p>
            <p>&copy; 2025 MDP System. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
		authorName,              // Hello, %s
		statusColor,             // border color
		statusIcon,              // status icon
		statusColor,             // status text color  
		status,                  // status text
		statusMessage,           // status message
		documentTitle,           // document title
		authorName,              // author name
		status,                  // status
		getReviewerInfo(reviewerName), // reviewer info (conditional)
		getCurrentTimestamp(),   // updated time
		getNextStepsHTML(status), // next steps based on status
	)

	// Set email body
	m.SetBody("text/html", htmlBody)

	// Create text alternative
	textBody := fmt.Sprintf(`
Document Status Update - MDP System

Hello %s,

Your document status has been updated:

Document Details:
- Title: %s
- Author: %s
- Status: %s
%s
- Updated: %s

%s

You can view your document in the MDP System dashboard.

© 2025 MDP System. All rights reserved.
`,
		authorName,
		documentTitle,
		authorName,
		status,
		getReviewerInfoText(reviewerName),
		getCurrentTimestamp(),
		getNextStepsText(status),
	)

	m.AddAlternative("text/plain", textBody)

	// Create SMTP dialer
	d := gomail.NewDialer(e.Host, e.Port, e.Username, e.Password)

	// Send email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send document status email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send document status email: %v", err)
	}

	log.Printf("Document status email sent successfully to %s (Document: %s, Status: %s)", toEmail, documentTitle, status)
	return nil
}

// Helper function to get reviewer information for HTML
func getReviewerInfo(reviewerName string) string {
	if reviewerName != "" {
		return fmt.Sprintf("<p><strong>Reviewer:</strong> %s</p>", reviewerName)
	}
	return ""
}

// Helper function to get reviewer information for text
func getReviewerInfoText(reviewerName string) string {
	if reviewerName != "" {
		return fmt.Sprintf("- Reviewer: %s", reviewerName)
	}
	return ""
}

// Helper function to get current timestamp
func getCurrentTimestamp() string {
	return time.Now().Format("January 2, 2006 at 3:04 PM")
}

// Helper function to get next steps based on status for HTML
func getNextStepsHTML(status string) string {
	switch status {
	case "In Review", "in_review", "review":
		return `
                    <li>Wait for reviewer feedback</li>
                    <li>Check your dashboard regularly for updates</li>
                    <li>Prepare for potential revisions based on feedback</li>`
	case "Approved", "approved":
		return `
                    <li>Your document is now live and accessible</li>
                    <li>Share the document link with stakeholders</li>
                    <li>Monitor document usage and feedback</li>`
	case "Rejected", "rejected":
		return `
                    <li>Review the feedback provided by the reviewer</li>
                    <li>Make necessary revisions to your document</li>
                    <li>Resubmit for review when ready</li>`
	case "Draft", "draft":
		return `
                    <li>Continue editing your document</li>
                    <li>Submit for review when complete</li>
                    <li>Save your progress regularly</li>`
	default:
		return `
                    <li>Check your dashboard for more details</li>
                    <li>Contact support if you have questions</li>`
	}
}

// Helper function to get next steps based on status for text
func getNextStepsText(status string) string {
	switch status {
	case "In Review", "in_review", "review":
		return `Next Steps:
- Wait for reviewer feedback
- Check your dashboard regularly for updates
- Prepare for potential revisions based on feedback`
	case "Approved", "approved":
		return `Next Steps:
- Your document is now live and accessible
- Share the document link with stakeholders  
- Monitor document usage and feedback`
	case "Rejected", "rejected":
		return `Next Steps:
- Review the feedback provided by the reviewer
- Make necessary revisions to your document
- Resubmit for review when ready`
	case "Draft", "draft":
		return `Next Steps:
- Continue editing your document
- Submit for review when complete
- Save your progress regularly`
	default:
		return `Next Steps:
- Check your dashboard for more details
- Contact support if you have questions`
	}
}

// ValidateEmailConfig checks if email configuration is properly set
// Similar to @PostConstruct validation in Spring Boot
func (e *EmailService) ValidateEmailConfig() error {
	if e.Host == "" {
		return fmt.Errorf("SMTP_HOST is not configured")
	}
	if e.Username == "" {
		return fmt.Errorf("SMTP_USERNAME is not configured")
	}
	if e.Password == "" {
		return fmt.Errorf("SMTP_PASSWORD is not configured")
	}
	if e.FromEmail == "" {
		return fmt.Errorf("SMTP_FROM_EMAIL is not configured")
	}
	return nil
}
