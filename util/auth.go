package util

import (
	db "api/database"
	"api/models"
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)


var jwtKey = []byte(db.PRIVKEY)
var tracer = otel.Tracer("utils")

// GenerateTokens generates the access and refresh tokens
func GenerateTokens(ctx context.Context, uuid string) (string, string) {
	ctx, span := tracer.Start(ctx, "GenerateTokens")
	accessToken, refreshToken := GenerateAccessClaims(ctx, uuid)
	span.AddEvent("Tokens are Generated.")
	time.Sleep(3 * time.Second)
	span.End()
	return accessToken, refreshToken
}

// GenerateAccessClaims returns a claim and a acess_token string
func GenerateAccessClaims(ctx context.Context, uuid string) (string, string) {
	ctx, span := tracer.Start(ctx, "AccessClaims")
	t := time.Now()
	claim := &models.Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    uuid,
			ExpiresAt: t.Add(1 * time.Hour).Unix(),
			Subject:   "access_token",
			IssuedAt:  t.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		panic(err)
	}

	span.SetAttributes(
		attribute.String("app.tokenString", tokenString),
	)
	span.AddEvent("Access claim is generated.")

	// Getting Refresh Token
	refreshToken := GenerateRefreshClaims(ctx, claim)
	span.End()
	return tokenString, refreshToken
}

// GenerateRefreshClaims returns refresh_token
func GenerateRefreshClaims(ctx context.Context, cl *models.Claims) string {
	_, span := tracer.Start(ctx, "RefreshClaims")
	dbCtx := db.DB.WithContext(ctx)
	result := dbCtx.Where(&models.Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer: cl.Issuer,
		},
	}).Find(&models.Claims{})

	// checking the number of refresh tokens stored.
	// If the number is higher than 3, remove all the refresh tokens and leave only new one.
	if result.RowsAffected > 3 {
		dbCtx.Where(&models.Claims{StandardClaims: jwt.StandardClaims{Issuer: cl.Issuer}}).Delete(&models.Claims{})
	}

	t := time.Now()
	refreshClaim := &models.Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    cl.Issuer,
			ExpiresAt: t.Add(10 * 24 * time.Hour).Unix(),
			Subject:   "refresh_token",
			IssuedAt:  t.Unix(),
		},
	}

	// create a claim on DB
	dbCtx.Create(&refreshClaim)

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaim)
	refreshTokenString, err := refreshToken.SignedString(jwtKey)
	if err != nil {
		panic(err)
	}
	span.AddEvent("Refres claim is generated.")
	span.End()
	return refreshTokenString
}

// SecureAuth returns a middleware which secures all the private routes
func SecureAuth() func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		accessToken := c.Get("access_token")
		claims := new(models.Claims)
		token, err := jwt.ParseWithClaims(accessToken, claims,
			func(token *jwt.Token) (interface{}, error) {
				return jwtKey, nil
			})

		if token.Valid {
			if claims.ExpiresAt < time.Now().Unix() {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   true,
					"general": "Token Expired",
				})
			}
		} else if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				// this is not even a token, we should delete the cookies here
				c.ClearCookie("access_token", "refresh_token")
				return c.SendStatus(fiber.StatusForbidden)
			} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				// Token is either expired or not active yet
				return c.SendStatus(fiber.StatusUnauthorized)
			} else {
				// cannot handle this token
				c.ClearCookie("access_token", "refresh_token")
				return c.SendStatus(fiber.StatusForbidden)
			}
		}

		c.Locals("id", claims.Issuer)
		return c.Next()
	}
}

// GetAuthCookies sends two cookies of type access_token and refresh_token
func GetAuthCookies(accessToken, refreshToken string) (*fiber.Cookie, *fiber.Cookie) {
	accessCookie := &fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
	}

	refreshCookie := &fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(10 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
	}

	return accessCookie, refreshCookie
}
