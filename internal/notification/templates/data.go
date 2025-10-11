package templates

// VerifyEmailData holds variables for the user.verify_email scenario using a 6-digit code.
type VerifyEmailData struct {
	FirstName    string
	Code         string
	SupportEmail string
}

// VerifyEmail is the typed handle for the user.verify_email template.
var VerifyEmail = Expect[VerifyEmailData]("user.verify_email")

// PasswordResetCodeData holds variables for sending a 6-digit password reset code.
type PasswordResetCodeData struct {
	FirstName    string
	Code         string
	SupportEmail string
}

// PasswordResetCode is the typed handle for the user.password_reset_code template.
var PasswordResetCode = Expect[PasswordResetCodeData]("user.password_reset_code")