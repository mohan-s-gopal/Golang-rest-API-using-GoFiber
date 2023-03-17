package controllers

import (
	db "api/database"
	"api/models"
	"api/util"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte(db.PRIVKEY)

// CreateUser route registers a User into the database
func CreateUser(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "Registration")
	defer span.End()

	u := new(models.User)

	if err := c.BodyParser(u); err != nil {
		print(err.Error())
		return c.JSON(Response{
			Error: true,
			Msg: "Please review your input",
		})
	}

	// validate if the email, username and password are in correct format
	errors := util.ValidateRegister(ctx, u)
	if errors.Err {
		return c.JSON(errors)
	}

	ctx, childSpan := tracer.Start(ctx, "checkEmailExists")
	if count := db.DB.WithContext(ctx).Where(&models.User{Email: u.Email}).First(new(models.User)).RowsAffected; count > 0 {
		errors.Err, errors.Msg = true, "Email is already registered"
	}
	childSpan.End()
	if errors.Err {
		return c.JSON(errors)
	}
	
	
	// ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	// defer cancel()

	ctx, usernameSpan := tracer.Start(ctx, "checkUsernameExists",)
	if count := db.DB.WithContext(ctx).Where(&models.User{Username: u.Username}).First(new(models.User)).RowsAffected; count > 0 {
		errors.Err, errors.Msg = true, "Username is already registered"
	}
	usernameSpan.End()
	if errors.Err {
		return c.JSON(errors)
	}
	

	// Hashing the password with a random salt
	ctx, getneratePasswordSpan := tracer.Start(ctx, "GenerateHashedPassword")
	password := []byte(u.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword(
		password,
		rand.Intn(bcrypt.MaxCost-bcrypt.MinCost)+bcrypt.MinCost,
	)
	getneratePasswordSpan.End()
	if err != nil {
		panic(err)
	}
	u.Password = string(hashedPassword)

	ctx, createUserSpan := tracer.Start(ctx, "CreateUser")
	if err := db.DB.WithContext(ctx).Create(&u).Error; err != nil {
		return c.JSON(Response{
			Error:   true,
			Msg: "Something went wrong, please try again later. 😕",
		})
	}
	createUserSpan.End()

	// setting up the authorization cookies
	accessToken, refreshToken := util.GenerateTokens(ctx, u.UUID.String())
	accessCookie, refreshCookie := util.GetAuthCookies(accessToken, refreshToken)
	c.Cookie(accessCookie)
	c.Cookie(refreshCookie)

	return c.Status(fiber.StatusOK).JSON(Response{
		Error: false, 
		Msg: "User created successfully",
	})
}

// LoginUser route logins a user in the app
func LoginUser(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "Login")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("http.method", c.Method()),
		attribute.String("http.url", c.Route().Name),
	)

	type LoginInput struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	 input := new(LoginInput)
	if err := c.BodyParser(input); err != nil {
		return c.JSON(Response{Error: true, Msg: "Please review your input"})
	}

	u := new(models.User)

	ctx, childSpan := tracer.Start(ctx, "checkEmailExists")
	if res := db.DB.WithContext(ctx).Where(
		&models.User{Email: input.Email}).First(&u); res.RowsAffected <= 0 {
		return c.JSON(Response{Error: true, Msg: "Invalid Credentials ."})
	}
	childSpan.End()

	// Comparing the password with the hash
	ctx, comparePasswordSpan := tracer.Start(ctx, "CompareHashAndPassword")
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(input.Password))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "internal error")
		return c.JSON(Response{Error: true, Msg: "Invalid Credentials."})
	}
	comparePasswordSpan.End()

	// setting up the authorization cookies
	accessToken, refreshToken := util.GenerateTokens(ctx, u.UUID.String())
	accessCookie, refreshCookie := util.GetAuthCookies(accessToken, refreshToken)

	c.Cookie(accessCookie)
	c.Cookie(refreshCookie)

	span.AddEvent("loggedin")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// GetAccessToken generates and sends a new access token iff there is a valid refresh token
func GetAccessToken(c *fiber.Ctx) error {
	type RefreshToken struct {
		RefreshToken string `json:"refresh_token"`
	}

	reToken := new(RefreshToken)
	if err := c.BodyParser(reToken); err != nil {
		return c.JSON(Response{Error: true, Msg: "Please review your input"})
	}

	refreshToken := reToken.RefreshToken

	refreshClaims := new(models.Claims)
	token, _ := jwt.ParseWithClaims(refreshToken, refreshClaims,
		func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

	if res := db.DB.Where(
		"expires_at = ? AND issued_at = ? AND issuer = ?",
		refreshClaims.ExpiresAt, refreshClaims.IssuedAt, refreshClaims.Issuer,
	).First(&models.Claims{}); res.RowsAffected <= 0 {
		// no such refresh token exist in the database
		c.ClearCookie("access_token", "refresh_token")
		return c.SendStatus(fiber.StatusForbidden)
	}

	if token.Valid {
		if refreshClaims.ExpiresAt < time.Now().Unix() {
			// refresh token is expired
			c.ClearCookie("access_token", "refresh_token")
			return c.SendStatus(fiber.StatusForbidden)
		}
	} else {
		// malformed refresh token
		c.ClearCookie("access_token", "refresh_token")
		return c.SendStatus(fiber.StatusForbidden)
	}

	_, accessToken := util.GenerateAccessClaims(c.UserContext(), refreshClaims.Issuer)

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
	})

	return c.JSON(fiber.Map{"access_token": accessToken})
}

/*
	PRIVATE ROUTES
*/

// GetUserData returns the details of the user signed in
func GetUserData(c *fiber.Ctx) error {
	id := c.Locals("id")
	_, span := tracer.Start(c.UserContext(), "GetUserData")
	defer span.End()

	u := new(models.User)
	if res := db.DB.Where("uuid = ?", id).First(&u); res.RowsAffected <= 0 {
		return c.JSON(Response{Error: true, Msg: "Cannot find the User"})
	}

	return c.JSON(u)
}
