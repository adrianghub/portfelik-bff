# Portfelik BFF (Backend for Frontend)

A Go backend service that connects to Firebase Admin SDK to provide API endpoints for the Portfelik frontend application.

## Features

- Firebase Authentication integration
- User profile management
- RESTful API endpoints
- Chi router for HTTP routing
- Graceful server shutdown
- Middleware for request logging and authentication

## Prerequisites

- Go 1.24+
- Firebase project with service account credentials
- Firebase CLI (for running emulators locally)

## Getting Started

1. Clone the repository:

```sh
git clone https://github.com/panizinko/portfelik-bff.git
cd portfelik-bff
```

2. Set up Firebase service account:

Copy your Firebase service account JSON file to `internal/config/service-account.json`

> Note: In development mode with emulators, you don't need a real service account file.

3. Configure environment variables:

There are two ways to configure environment variables:
- Copy `.env.example` to `.env` in the root directory
- Update the `internal/config/.env` file

For development with emulators, use:

```
# Server settings
PORT=8888

# Firebase Emulator settings
GO_ENV=development
FIREBASE_AUTH_EMULATOR_HOST=localhost:9099
FIRESTORE_EMULATOR_HOST=localhost:8080
FIREBASE_PROJECT_ID=portfelik-dev
```

4. Build and run the application:

```sh
go run main.go
```

## Testing with Firebase Emulators

This service is designed to work with Firebase emulators for local development.

1. Start the Firebase emulators:

```sh
firebase emulators:start --only auth,firestore
```

2. Start the BFF service:

```sh
go run main.go
```

3. Create a test user and get a token:

```sh
# Create a user and get a token
./scripts/get_token.sh --create

# Get a token for an existing user
./scripts/get_token.sh

# Get a token and test the API
./scripts/get_token.sh --test
```

The script provides options for email, password, emulator host, and API host:

```sh
./scripts/get_token.sh --email user@example.com --password secret123 --create --test
```

## API Endpoints

### Public Endpoints

- `GET /health`: Health check endpoint

### Protected Endpoints (Require Firebase Authentication)

- `GET /api/v1/users/me`: Get current user profile
- `GET /api/v1/users/{uid}`: Get user profile by ID

## Authentication

The API uses Firebase Authentication with JWT tokens. To authenticate requests, include an `Authorization` header with a Bearer token:

```
Authorization: Bearer <firebase-id-token>
```

## Project Structure

```
portfelik-bff/
├── main.go                 # Application entry point
├── .env                    # Root environment variables
├── scripts/                # Utility scripts
│   └── get_token.sh        # Token retrieval script
├── internal/
│   ├── auth/               # Authentication service
│   ├── config/             # Configuration files
│   ├── handlers/           # HTTP handlers
│   ├── logger/             # Logging functionality
│   ├── middleware/         # HTTP middleware
│   └── models/             # Data models
```

## License

MIT

## Cloud Run Deployment

### Manual Deployment

To deploy the service to Google Cloud Run manually:

1. Install the Google Cloud SDK:
   ```sh
   brew install google-cloud-sdk  # macOS with Homebrew
   ```

2. Authenticate with Google Cloud:
   ```sh
   gcloud auth login
   gcloud config set project YOUR_GCP_PROJECT_ID
   ```

3. Deploy to Cloud Run:
   ```sh
   gcloud run deploy portfelik-bff --source . --region us-central1 --platform managed --allow-unauthenticated
   ```

4. Set environment variables for production:
   ```sh
   gcloud run services update portfelik-bff \
     --set-env-vars="GO_ENV=production,FIREBASE_PROJECT_ID=your-project-id" \
     --region us-central1
   ```

### Automated Deployment

This repository includes two ways to set up continuous deployment:

1. **GitHub Actions**: Automatically deploys when changes are pushed to the main branch.
   - Required secrets:
     - `GCP_PROJECT_ID`: Your Google Cloud project ID
     - `GCP_SA_KEY`: JSON credentials for a service account with Cloud Run Admin permissions
     - `FIREBASE_SA_JSON`: Firebase service account JSON for authentication
     - `FIREBASE_PROJECT_ID`: Your Firebase project ID

2. **Cloud Build**: Use Google Cloud Build for CI/CD pipeline.
   - To set up a Cloud Build trigger:
     ```sh
     gcloud builds triggers create github \
       --repo=your-github-repo \
       --branch-pattern=main \
       --build-config=cloudbuild.yaml
     ```
   - Set the required substitution variables in the Cloud Build trigger settings or modify them in the cloudbuild.yaml file.

### Managing Service Accounts

For deployment to work correctly, you need:

1. A GCP service account with these roles:
   - Cloud Run Admin
   - Storage Admin
   - Service Account User

2. A Firebase service account with Firebase Admin SDK access
   - Download the JSON file and keep it secure
   - For GitHub Actions, store it as a secret
   - For manual deployment, save it at `internal/config/service-account.json`

### Accessing Your Deployed Service

Once deployed, your service will be available at:
```
https://portfelik-bff-[hash].a.run.app
```

You can get the URL after deployment is complete using:
```sh
gcloud run services describe portfelik-bff --platform managed --region us-central1 --format 'value(status.url)'
```