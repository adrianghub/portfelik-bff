package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type FirebaseAuth struct {
	client *auth.Client
}

func NewFirebaseAuth(credentialsPath string) (*FirebaseAuth, error) {
	ctx := context.Background()

	emulatorHost := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if emulatorHost != "" {
		log.Printf("Using Firebase Auth emulator at %s", emulatorHost)

		// When using the emulator, there is no need for real credentials
		app, err := firebase.NewApp(ctx, &firebase.Config{
			ProjectID: os.Getenv("FIREBASE_PROJECT_ID"),
		})
		if err != nil {
			return nil, fmt.Errorf("error initializing Firebase app for emulator: %v", err)
		}

		client, err := app.Auth(ctx)
		if err != nil {
			return nil, fmt.Errorf("error initializing Firebase auth client for emulator: %v", err)
		}

		return &FirebaseAuth{
			client: client,
		}, nil
	}

	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase app: %v", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase auth client: %v", err)
	}

	return &FirebaseAuth{
		client: client,
	}, nil
}

func (fa *FirebaseAuth) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	token, err := fa.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("error verifying ID token: %v", err)
	}
	return token, nil
}

func (fa *FirebaseAuth) GetUserByUID(ctx context.Context, uid string) (*auth.UserRecord, error) {
	user, err := fa.client.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error getting user by UID: %v", err)
	}
	return user, nil
}

func (fa *FirebaseAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			idToken := strings.TrimPrefix(authHeader, "Bearer ")
			if idToken == authHeader {
				http.Error(w, "Invalid authorization format, expected 'Bearer <token>'", http.StatusUnauthorized)
				return
			}

			token, err := fa.VerifyIDToken(r.Context(), idToken)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			type userContextKey struct{}

			ctx := context.WithValue(r.Context(), userContextKey{}, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
