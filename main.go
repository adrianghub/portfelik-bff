package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/panizinko/portfelik-bff/internal/auth"
	"github.com/panizinko/portfelik-bff/internal/handlers"
	"github.com/panizinko/portfelik-bff/internal/logger"
	"github.com/panizinko/portfelik-bff/internal/middleware"
	"google.golang.org/api/option"
)

func main() {
	err := godotenv.Load("internal/config/.env")
	_ = godotenv.Load()

	if err != nil {
		log.Printf("Warning: Error loading .env file from config directory: %v", err)
	}

	logger := logger.NewLogger()
	logger.Info("Starting Portfelik BFF API")

	isDevelopment := os.Getenv("GO_ENV") == "development"
	if isDevelopment {
		logger.Info("Running in development mode")

		if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") == "" {
			os.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "localhost:9099")
			logger.Info("Using Firebase Auth emulator at localhost:9099")
		}

		if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
			os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
			logger.Info("Using Firestore emulator at localhost:8080")
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	var firebaseApp *firebase.App
	var authClient *firebaseauth.Client
	var credentialsPath string

	if isDevelopment {
		firebaseConfig := &firebase.Config{
			ProjectID: os.Getenv("FIREBASE_PROJECT_ID"),
		}

		if firebaseConfig.ProjectID == "" {
			firebaseConfig.ProjectID = "portfelik-dev"
			logger.Info("Using default project ID: %s", firebaseConfig.ProjectID)
		}

		app, err := firebase.NewApp(context.Background(), firebaseConfig)
		if err != nil {
			logger.Error("Error initializing Firebase app: %v", err)
			os.Exit(1)
		}

		firebaseApp = app
	} else {
		credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if credentialsPath == "" {
			logger.Error("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
			os.Exit(1)
		}

		opt := option.WithCredentialsFile(credentialsPath)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			logger.Error("Error initializing Firebase app: %v", err)
			os.Exit(1)
		}

		firebaseApp = app
	}

	authClient, err = firebaseApp.Auth(context.Background())
	if err != nil {
		logger.Error("Error initializing Firebase auth client: %v", err)
		os.Exit(1)
	}

	authService, err := auth.NewFirebaseAuth(credentialsPath)
	if err != nil {
		logger.Error("Error initializing Firebase auth service: %v", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authService.Middleware())

			userHandler := handlers.NewUserHandler(authClient, logger)
			r.Mount("/users", userHandler.Routes())
		})
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Error("Graceful shutdown timed out... forcing exit.")
				os.Exit(1)
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			logger.Error("Error during server shutdown: %v", err)
		}
		serverStopCtx()
	}()

	logger.Info("Server listening on port %s", port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error("Error starting server: %v", err)
		os.Exit(1)
	}

	<-serverCtx.Done()
	logger.Info("Server stopped")
}
