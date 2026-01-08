package dto

import (
	"regexp"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/golang-jwt/jwt/v5"
)

var (
	Validate *validator.Validate
	Trans    ut.Translator
)

func InitValidator() {
	en := en.New()
	uni := ut.New(en, en)
	Trans, _ = uni.GetTranslator("en")

	Validate = validator.New()

	_ = enTranslations.RegisterDefaultTranslations(Validate, Trans)

	_ = Validate.RegisterValidation("trim", trimValue) // SIDE EFFECT: trims the value
	_ = Validate.RegisterValidation("username", validateUsername)
	_ = Validate.RegisterValidation("password", validatePassword)
	_ = Validate.RegisterValidation("identifier", validateIdentifier)
	registerUsernameTranslation(Validate, Trans)
	registerPasswordTranslation(Validate, Trans)
	registerIdentifierTranslation(Validate, Trans)
}

// Space Trimming, SIDE EFFECT!
func trimValue(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	trimed := strings.TrimSpace(value)
	fl.Field().SetString(trimed)

	return true
}

// Username

type UserName struct {
	Username string `json:"username" validate:"required,trim,min=3,max=50,username"`
}

// Contains only letters, numbers, ".", "_" or "-"
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	return usernameRegex.MatchString(username)
}

func registerUsernameTranslation(v *validator.Validate, trans ut.Translator) {
	_ = v.RegisterTranslation(
		"username",
		trans,
		func(ut ut.Translator) error {
			return ut.Add(
				"username",
				"username may only contain letters, numbers, '.', '_' or '-'",
				true,
			)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			msg, _ := ut.T("username")
			return msg
		},
	)
}

// Password

type Password struct {
	Password string `json:"password" validate:"required,trim,min=3,max=20,password"`
}

type OldPassword struct {
	OldPassword string `json:"oldPassword" validate:"required,trim,password,min=3,max=20"`
}

type NewPassword struct {
	NewPassword string `json:"newPassword" validate:"required,trim,password,min=3,max=20"`
}

// Contains only letters, numbers, ".", "_" or "-"
var passwordRegex = regexp.MustCompile(`^[A-Za-z0-9,.#$%@^;|_!*&?]+$`)

func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	return passwordRegex.MatchString(password)
}

func registerPasswordTranslation(v *validator.Validate, trans ut.Translator) {
	_ = v.RegisterTranslation(
		"password",
		trans,
		func(ut ut.Translator) error {
			return ut.Add(
				"password",
				"password may only contain letters, numbers, and the following symbols: ,.#$%@^;|_!*&?",
				true,
			)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			msg, _ := ut.T("password")
			return msg
		},
	)
}

// Identifier
type Identifier struct {
	Identifier string `json:"identifier" validate:"required,trim,min=3,max=100,identifier"` // username or email
}

func validateIdentifier(fl validator.FieldLevel) bool {
	identifier := fl.Field().String()

	usernameErrs := Validate.Var(identifier, "username")
	emailErrs := Validate.Var(identifier, "email")

	if usernameErrs == nil || emailErrs == nil {
		return true
	}

	return false
}

func registerIdentifierTranslation(v *validator.Validate, trans ut.Translator) {
	_ = v.RegisterTranslation(
		"identifier",
		trans,
		func(ut ut.Translator) error {
			return ut.Add(
				"identifier",
				"identifier may only contain a valid username or email address",
				true,
			)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			msg, _ := ut.T("identifier")
			return msg
		},
	)
}

// User DTOs

type User struct {
	UserName
	Email  string  `json:"email" validate:"required,trim,email,max=100"`
	Avatar *string `json:"avatar" validate:"omitempty,url"`
}

type SimpleUser struct {
	ID       uint    `json:"id"`
	Username string  `json:"username"`
	Avatar   *string `json:"avatar"`
}

type CreateUserRequest struct {
	User
	Password
}

type UpdateUserPasswordRequest struct {
	OldPassword
	NewPassword
}

type LoginUserRequest struct {
	Identifier
	Password
}

type UpdateUserRequest struct {
	User
}

type UsernameRequest struct {
	UserName
}

type UserWithTokenResponse struct {
	ID            uint    `json:"id"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	Avatar        *string `json:"avatar"`
	TwoFA         bool    `json:"twoFa"`
	GoogleOauthId *string `json:"googleOauthId,omitempty"`
	CreatedAt     int64   `json:"createdAt"`
	Token         string  `json:"token"`
}

type UserWithoutTokenResponse struct {
	ID            uint    `json:"id"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	Avatar        *string `json:"avatar"`
	TwoFA         bool    `json:"twoFa"`
	GoogleOauthId *string `json:"googleOauthId,omitempty"`
	CreatedAt     int64   `json:"createdAt"`
}

type UsersResponse struct {
	Users []SimpleUser `json:"users"`
}

// For 2FA

type SetTwoFARequest struct {
	TwoFA bool `json:"twoFa" validate:"required"`
}

type DisableTwoFARequest struct {
	Password
}

type TwoFAConfirmRequest struct {
	TwoFACode  string `json:"twoFaCode" validate:"required,len=6,numeric"`
	SetupToken string `json:"setupToken" validate:"required"`
}

type TwoFAChallengeRequest struct {
	TwoFACode    string `json:"twoFaCode" validate:"required,len=6,numeric"`
	SessionToken string `json:"sessionToken" validate:"required"`
}

type TwoFASetupResponse struct {
	TwoFASecret string `json:"twoFaSecret" validate:"required"`
	SetupToken  string `json:"setupToken" validate:"required"`
}

type TwoFAPendingUserResponse struct {
	Message      string `json:"message"`
	SessionToken string `json:"sessionToken"`
}

type AddNewFriendRequest struct {
	UserID uint `json:"userId" validate:"required"`
}

type FriendResponse struct {
	SimpleUser
	Online bool `json:"online"`
}

type FriendsResponse struct {
	Friends []FriendResponse `json:"friends"`
}

type UserValidationResponse struct {
	UserID uint `json:"userId"`
}

type GoogleOauthCallback struct {
	Code  string `json:"code" validate:"required"`
	State string `json:"state" validate:"required"`
}

type GoogleUserData struct {
	ID      string  `json:"id"`
	Email   string  `json:"email"`
	Name    string  `json:"name"`
	Picture *string `json:"picture"`
}

type GoogleJwtPayload struct {
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type GoogleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type UserJwtPayload struct {
	UserID uint   `json:"userId"`
	Type   string `json:"type"` // must be "USER"
	jwt.RegisteredClaims
}

type OauthStateJwtPayload struct {
	Type string `json:"type"` // must be "GoogleOAuthState"
	jwt.RegisteredClaims
}

type TwoFaSetupJwtPayload struct {
	UserID uint   `json:"userId"`
	Secret string `json:"secret"`
	Type   string `json:"type"` // must be "2FA_SETUP"
	jwt.RegisteredClaims
}

type TwoFaJwtPayload struct {
	UserID uint   `json:"userId"`
	Type   string `json:"type"` // must be "2FA"
	jwt.RegisteredClaims
}
