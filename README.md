# LivePoll – Real-Time Interactive Polling Platform

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Gin Framework](https://img.shields.io/badge/Gin-v1.10.0-008ECF?style=flat&logo=go)](https://gin-gonic.com)
[![React](https://img.shields.io/badge/React-18.3-61DAFB?style=flat&logo=react)](https://react.dev)
[![Vite](https://img.shields.io/badge/Vite-6.0-646CFF?style=flat&logo=vite)](https://vitejs.dev)
[![MongoDB](https://img.shields.io/badge/MongoDB-7.0-47A248?style=flat&logo=mongodb)](https://www.mongodb.com)
[![Redis](https://img.shields.io/badge/Redis-7.2-DC382D?style=flat&logo=redis)](https://redis.io)
[![WebSocket](https://img.shields.io/badge/WebSocket-Gorilla-blue?style=flat)](https://github.com/gorilla/websocket)

> Built for the **GUVI HCL Developer Internship Evaluation Task**.  
> Engineered with a high-concurrency Go (Gin) backend, permanent MongoDB persistence, Redis Pub/Sub event bus, and an animated React (Vite) frontend with true zero-refresh live updates.

---

## Table of Contents
1. [Problem Statement](#problem-statement)
2. [Architectural Overview](#architectural-overview)
3. [Tech Stack & Responsibilities](#tech-stack--responsibilities)
4. [Project Structure](#project-structure)
5. [Key Features](#key-features)
6. [Admin View & Linked MongoDB Dashboard](#admin-view--linked-mongodb-dashboard)
7. [API Endpoints Reference](#api-endpoints-reference)
8. [Getting Started (Step-by-Step)](#getting-started-step-by-step)
   - [Prerequisites](#prerequisites)
   - [Option A: Docker Compose (Quickest)](#option-a-docker-compose-quickest)
   - [Option B: Native Local Installation](#option-b-native-local-installation)
   - [Option C: Free Cloud Databases (Atlas + Upstash)](#option-c-free-cloud-databases-atlas--upstash)
9. [Running the Application](#running-the-application)
10. [Real-Time Multi-Browser Verification](#real-time-multi-browser-verification)
11. [Postman API Testing](#postman-api-testing)
12. [Git & Repository Setup](#git--repository-setup)
13. [Live Production Deployment](#live-production-deployment)
14. [Submission Video Walkthrough Guide (Mandatory)](#submission-video-walkthrough-guide-mandatory)
15. [Technical Interview Preparation (Q&A)](#technical-interview-preparation-qa)

---

## Problem Statement

Build a live polling tool where a user can create a poll, share a unique link with an audience, and allow multiple viewers to vote simultaneously. Everyone watching the poll must see the results update dynamically in real time **without refreshing the page**.

### The Core Flow
$$\text{Create Poll} \longrightarrow \text{Share Unique Link} \longrightarrow \text{Audience Votes} \longrightarrow \text{Real-Time Results Stream}$$

---

## Architectural Overview

LivePoll strictly enforces the real-time pipeline where **Redis Pub/Sub** acts as the event broker and **MongoDB** acts as the permanent document store:

```
[ Audience Browser 2 ] 
          │
          │ 1. POST /api/polls/:code/vote { optionId: "opt-1" }
          ▼
[ Go + Gin REST API ]
          │
          ├── 2. Input Validation (Checks code, option, poll status)
          │
          ├── 3. MongoDB: Atomic $inc operation
          │      Filter: { code: "abc123", "options.id": "opt-1" }
          │      Update: { $inc: { "options.$.votes": 1, totalVotes: 1 } }
          │
          └── 4. Redis Pub/Sub: Publish Event
                 Channel: "poll:abc123"
                 Payload: { type: "VOTE_UPDATE", poll: <updatedData> }
                           │
                           ▼
                 [ Redis Pub/Sub Engine ]
                           │
                           ▼ (Broadcast to subscribers)
                 [ Go WebSocket Hub Room ]
                           │
                           │ 5. Transmit WebSocket Message Frame
                           ▼
          [ Audience Browser 1 (Live Results View) ]
          🎉 Results animate instantly (NO PAGE REFRESH!)
```

---

## Tech Stack & Responsibilities

| Layer | Technology | Real Responsibility |
| :--- | :--- | :--- |
| **Frontend** | React 18, Vite, React Router, Lucide Icons, Canvas Confetti | Modern responsive UI, micro-animations, animated progress bars, live radar ping indicator, client-side WebSocket client with exponential backoff auto-reconnect. |
| **Backend** | Go (Golang 1.22+), Gin Engine, Gorilla WebSocket | High-performance RESTful APIs, JWT authentication, bcrypt password hashing, input validation, concurrency-safe WebSocket Hub room management. |
| **Database** | MongoDB 7.0 | Permanent ACID document store for users, polls, options, vote counts, creator IDs, unique indexes on email and poll code. Atomic updates via `$inc`. |
| **Realtime** | Redis 7.2 | Message broker with Pub/Sub. Votes are published to channel `poll:{code}`. Decouples vote ingestion from WebSocket distribution across instances. |

---

## Project Structure

Frontend and backend codebases are completely separated:

```
livepoll/
├── backend/
│   ├── config/
│   │   ├── config.go          # Environment variable parser
│   │   ├── mongo.go           # MongoDB client connection & unique index creation
│   │   └── redis.go           # Redis client initialization & publish helpers
│   ├── handlers/
│   │   ├── admin.go           # MongoDB manual dashboard & system diagnostics
│   │   ├── auth.go            # User signup & login with bcrypt & JWT
│   │   ├── poll.go            # Poll creation, public retrieval, user polls, deletion
│   │   └── vote.go            # Atomic vote increments and Redis publishing
│   ├── middleware/
│   │   ├── auth.go            # JWT Bearer token authentication & claims extraction
│   │   └── cors.go            # CORS middleware allowing React origin
│   ├── models/
│   │   ├── poll.go            # Poll, Option, and WebSocket payload structs
│   │   └── user.go            # User struct & Auth request/response DTOs
│   ├── routes/
│   │   └── routes.go          # Gin routing tree & endpoint registration
│   ├── websocket/
│   │   ├── client.go          # WebSocket client read/write pumps & ping-pong heartbeat
│   │   └── hub.go             # Global WebSocket Hub & Redis Pub/Sub channel listener
│   ├── .env.example           # Environment template
│   ├── go.mod                 # Go module definition
│   ├── go.sum                 # Cryptographic checksums
│   └── main.go                # Application bootstrap & server listener
│
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Footer.jsx     # Tech stack credits & status badges
│   │   │   ├── LiveBadge.jsx  # Pulsing live radar indicator
│   │   │   ├── Navbar.jsx     # Navigation bar with role badges & session controls
│   │   │   ├── ProgressBar.jsx# Spring-animated vote percentage progress bar
│   │   │   ├── ShareModal.jsx # One-click copy link & QR sharing modal
│   │   │   └── Toast.jsx      # Animated toast notifications
│   │   ├── context/
│   │   │   └── AuthContext.jsx# Persistent JWT auth session provider
│   │   ├── pages/
│   │   │   ├── AdminDB.jsx    # MongoDB Manual Dashboard & Collection Explorer
│   │   │   ├── CreatePoll.jsx # Dynamic poll builder with option addition/removal
│   │   │   ├── Dashboard.jsx  # User control hub with metrics & recent polls
│   │   │   ├── Landing.jsx    # Hero landing page & quick join by code
│   │   │   ├── LiveResults.jsx# Real-time WebSocket live results with presenter mode
│   │   │   ├── Login.jsx      # JWT login form with validation
│   │   │   ├── MyPolls.jsx    # Poll management with search & deletion
│   │   │   ├── NotFound.jsx   # 404 page for invalid poll codes
│   │   │   ├── Signup.jsx     # User registration form
│   │   │   └── VotePoll.jsx   # Audience voting screen with confetti animation
│   │   ├── services/
│   │   │   ├── api.js         # Centralized HTTP client with auto-attached JWT
│   │   │   └── websocket.js   # Resilient WebSocket connection manager
│   │   ├── App.jsx            # Routing hierarchy & protected route guards
│   │   ├── index.html         # HTML entry point with Plus Jakarta Sans typography
│   │   ├── main.jsx           # React DOM root mounting
│   │   └── style.css          # Glassmorphism, CSS variables & keyframe animations
│   ├── .env.example           # Frontend environment template
│   ├── package.json           # Node dependencies
│   └── vite.config.js         # Vite configuration with API proxy
│
├── docker-compose.yml         # MongoDB, Redis, and Mongo Express orchestration
├── LivePoll.postman_collection.json # Automated API test suite
├── .gitignore                 # Excludes secrets, node_modules, and binaries
└── README.md                  # Comprehensive developer & evaluator guide
```

---

## Key Features

1. **User Authentication & Security**:
   - Secure registration and login using JWT (HMAC-SHA256).
   - Password encryption using **bcrypt** (salt cost 10). Passwords are never serialized into JSON.
   - Protected endpoints require valid Bearer token.
2. **Interactive Poll Creation**:
   - Dynamic option builder (starts with 2, add up to 10 options, remove options).
   - Server-side validation: minimum question length, trimmed strings, prevention of duplicate choices.
   - Nano-code generation producing clean 6-character share codes (e.g. `abc123`).
3. **Audience Voting (No Login Required)**:
   - Audience members can access any poll via `/poll/:code`.
   - Responsive radio option cards with smooth selection feedback.
   - Local double-voting prevention via `localStorage`.
   - Festive confetti burst animation upon voting!
4. **Zero-Refresh Live Results**:
   - Connects to `/api/polls/:code/live` over Gorilla WebSockets.
   - Sends initial snapshot immediately on connection (`INIT`).
   - Automatically animates vote percentages and counters upon `VOTE_UPDATE`.
   - Pulsing live badge with radar ping animation.
   - Fullscreen presenter mode for project screens and conferences.
5. **Poll Management**:
   - Creators can browse their polls, view total votes across all polls, search by title or code, copy share links, or delete polls with automatic live viewer notification.

---

## Admin View & Linked MongoDB Dashboard

To fulfill the developer evaluation requirement for direct database visibility, LivePoll includes an **Admin View & MongoDB Manual Dashboard** directly within the web app at `/admin/db`:

- **Real-Time Diagnostics**:
  - Live MongoDB connection state, database name (`livepoll`), and ping latency in milliseconds.
  - Live Redis connection state and address (`localhost:6379`).
  - Total document counts for the `users` and `polls` collections.
- **Raw Document Inspector**:
  - Browse every document stored inside MongoDB.
  - View full JSON documents (with sensitive passwords scrubbed).
- **External Tools Integration**:
  - Built-in guidance for connecting via **MongoDB Compass** (`mongodb://localhost:27017`).
  - Web UI via **Mongo Express** at `http://localhost:8081` (included in `docker-compose.yml`).
  - Terminal stream monitor via `redis-cli PSUBSCRIBE "poll:*"`.

---

## API Endpoints Reference

### Public Authentication
- `POST /api/signup` — Register a new account (`name`, `email`, `password`)
- `POST /api/login` — Authenticate and receive JWT token (`email`, `password`)

### Public Audience Endpoints
- `GET /api/polls/:code` — Retrieve poll question and choices by share code
- `POST /api/polls/:code/vote` — Submit a vote (`optionId`). Atomically updates MongoDB and publishes to Redis Pub/Sub.
- `GET /api/polls/:code/live` — WebSocket endpoint for real-time live poll streaming.

### Protected Creator Endpoints (Requires `Authorization: Bearer <token>`)
- `POST /api/polls` — Create a new poll (`question`, `options[]`)
- `GET /api/my-polls` — Retrieve all polls created by the authenticated user
- `DELETE /api/polls/:id` — Delete a poll (verifies ownership before deletion)

### System & Diagnostics
- `GET /api/health` — Checks MongoDB and Redis connection status
- `GET /api/admin/db-stats` — Returns database metadata, latency, and collection summaries
- `GET /api/admin/collections/:name` — Returns stored documents for the manual dashboard

---

## Getting Started (Step-by-Step)

### Prerequisites
- **Node.js** (v18 or higher) — [Download Node.js](https://nodejs.org)
- **Go** (v1.21 or higher) — [Download Go](https://go.dev)
- **Docker** (Recommended) or local MongoDB & Redis instances.

---

### Option A: Docker Compose (Quickest & Easiest)

Start MongoDB, Redis, and Mongo Express with a single command:

```bash
docker compose up -d
```

This starts:
- **MongoDB** on `localhost:27017`
- **Redis** on `localhost:6379`
- **Mongo Express Web GUI** on `http://localhost:8081` (User: `admin`, Pass: `admin`)

To verify containers are running:
```bash
docker ps
```

---

### Option B: Native Local Installation

#### 1. MongoDB Community Server
- Download and install [MongoDB Community Server](https://www.mongodb.com/try/download/community).
- Start the MongoDB service:
  ```powershell
  # Windows PowerShell (Run as Administrator)
  net start MongoDB
  ```

#### 2. Redis
- On Windows, install Redis via [Redis for Windows](https://github.com/microsoftarchive/redis/releases) or use WSL2 (`sudo apt install redis-server && sudo service redis-server start`).
- Verify Redis is running:
  ```bash
  redis-cli ping
  # Expected output: PONG
  ```

---

### Option C: Free Cloud Databases (Atlas + Upstash)

If you do not want to install databases locally, LivePoll works seamlessly with free cloud database providers:

1. **MongoDB Atlas** (Free M0 Sandbox):
   - Sign up at [mongodb.com/cloud/atlas](https://www.mongodb.com/cloud/atlas).
   - Create a free shared cluster.
   - Copy the connection URI: `mongodb+srv://<username>:<password>@cluster0.mongodb.net/livepoll?retryWrites=true&w=majority`
   - Paste it as `MONGO_URI` in `backend/.env`.

2. **Upstash Redis** (Free Serverless Redis):
   - Sign up at [upstash.com](https://upstash.com).
   - Create a free Redis database.
   - Copy the Endpoint (`xxx.upstash.io:6379`) and Password.
   - Paste them as `REDIS_ADDR` and `REDIS_PASSWORD` in `backend/.env`.

---

## Running the Application

### 1. Start the Go Backend

Open a terminal in the `backend` directory:

```bash
cd backend

# Download Go dependencies (first time only)
go mod tidy

# Run the server
go run main.go
```

The Go backend will output:
```
🚀 Initializing LivePoll Backend Service...
Connected to MongoDB at mongodb://localhost:27017 (Database: livepoll)
Connected to Redis at localhost:6379
📡 Real-time WebSocket Hub started
LivePoll API Server listening on port :8080
```

---

### 2. Start the React Frontend

Open a second terminal in the `frontend` directory:

```bash
cd frontend

# Install npm packages (first time only)
npm install

# Start Vite development server
npm run dev
```

The frontend will start on:
```
➜  Local:   http://localhost:5173/
```

Open [http://localhost:5173](http://localhost:5173) in Google Chrome!

---

## Real-Time Multi-Browser Verification

To verify that the system is **truly real-time** with zero page refreshes:

1. Open **Browser Window 1** (or a Chrome Incognito window):
   - Navigate to [http://localhost:5173/login](http://localhost:5173/login).
   - Sign in or create an account, then click **Create Poll**.
   - Enter a question (e.g., *"What is your preferred cloud stack?"*) and options (*"Go + AWS"*, *"Node + GCP"*, *"Python + Azure"*).
   - Click **Launch Live Poll**.
   - Navigate to the **Live Stream** page (`/poll/abc123/results`). Notice the green **"LIVE UPDATES"** pulsing radar badge.

2. Open **Browser Window 2**:
   - Paste the audience voting link (`/poll/abc123`).
   - Select one of the options and click **Submit Vote**.

3. **Observe**:
   - In **Browser Window 1**, the vote count and animated progress bar will update **instantly** without touching the keyboard or refreshing the page!
   - Redis Pub/Sub receives the vote event from Go and pushes it directly down the WebSocket connection to Browser 1 in less than 50ms.

---

## Postman API Testing

The repository includes `LivePoll.postman_collection.json`.

1. Open **Postman**.
2. Click **Import** and select `LivePoll.postman_collection.json`.
3. Run the requests in numerical order:
   - **System Health**: Asserts MongoDB & Redis are active.
   - **1. User Signup**: Creates an evaluator account and auto-saves the JWT token.
   - **2. User Login**: Verifies credentials and token refresh.
   - **3. Create Poll**: Creates a live poll and auto-saves `pollCode` and `optionId`.
   - **4. Get Poll by Code**: Verifies public retrieval of poll choices.
   - **5. Cast Vote**: Simulates an audience vote, atomically incrementing MongoDB and triggering Redis broadcast.
   - **6. Get My Polls**: Verifies creator poll listing and vote aggregation.
   - **7. DB Stats & Redis Diagnostic**: Validates database latency and document counts.
   - **8. Inspect MongoDB Documents**: Reads collection records directly.
   - **9. Delete Poll**: Cleans up test poll and notifies active WebSocket viewers.

---

## Git & Repository Setup

To commit and push this project to your GitHub account:

```bash
# 1. Initialize git repository
git init

# 2. Stage all project files (sensitive .env and node_modules are excluded via .gitignore)
git add .

# 3. Create the initial commit
git commit -m "feat: complete live polling platform with Go, Gin, Redis, MongoDB, and React"

# 4. Set default branch to main
git branch -M main

# 5. Add your remote GitHub repository
git remote add origin https://github.com/<YOUR_USERNAME>/live-polling-tool.git

# 6. Push code to GitHub
git push -u origin main
```

---

## Live Production Deployment

### 1. Frontend Deployment (Vercel or Netlify)
- Link your GitHub repository to **Vercel** or **Netlify**.
- Set the Root Directory to `frontend`.
- Build Command: `npm run build`
- Output Directory: `dist`
- Environment Variable:
  - `VITE_API_URL` = `https://<your-backend-domain>.onrender.com/api`

### 2. Backend Deployment (Render or Railway)
- Create a new **Web Service** on [Render.com](https://render.com) or [Railway.app](https://railway.app).
- Root Directory: `backend`
- Environment: `Go`
- Build Command: `go build -o server main.go`
- Start Command: `./server`
- Environment Variables:
  - `PORT` = `8080`
  - `MONGO_URI` = `<Your MongoDB Atlas Connection String>`
  - `REDIS_ADDR` = `<Your Upstash Redis Host:Port>`
  - `REDIS_PASSWORD` = `<Your Upstash Redis Password>`
  - `JWT_SECRET` = `<Your Secure Random String>`
  - `CLIENT_URL` = `https://<your-frontend>.vercel.app`

---

## Submission Video Walkthrough Guide (Mandatory)

Per the internship instructions, a **3–5 minute video** (unlisted YouTube or public Google Drive link) must be submitted to `devhiring@hclguvi.com`.

### Suggested Video Structure:
1. **Introduction (0:00 - 0:45)**:
   - Introduce yourself and summarize the project goal: A real-time polling tool with Go, Gin, MongoDB, Redis, and React.
   - Demonstrate the multi-browser live voting flow (Window 1 showing results, Window 2 casting vote, results updating with no refresh).
2. **Architecture Walkthrough (0:45 - 2:00)**:
   - Explain how Go handles the HTTP request, performs atomic `$inc` updates on MongoDB, and publishes the event to Redis channel `poll:{code}`.
   - Show how the Go WebSocket Hub room subscribes to Redis Pub/Sub and dispatches frames to connected clients.
   - Demonstrate the **MongoDB Manual Dashboard** (`/admin/db`) showing real-time document counts, database latency, and stored records.
3. **The Challenge Faced & Solution (2:00 - 3:15)** *(Mandatory Question)*:
   - **Challenge**: *"Ensuring that multiple WebSocket viewers connected to the same poll room receive instant updates efficiently without overwhelming Redis with separate connections for every single client."*
   - **Solution**: *"I designed a centralized WebSocket Hub in Go that maintains a map of active poll rooms. When the first viewer connects to a poll, Go opens a single Redis Pub/Sub subscription for that poll channel. When a vote arrives, Redis publishes once, and Go distributes it concurrently to all connected WebSocket clients in that room. When all viewers leave, Go automatically cancels the context and unsubscribes from Redis, preventing memory leaks and optimizing connection usage."*
4. **AI Tool Transparency (3:15 - 4:00)** *(Mandatory Question)*:
   - Be clear and honest: *"I leveraged AI coding assistants to scaffold boilerplate code, assist with CSS spring animations, and draft Docker Compose configurations. However, I took full responsibility for the architectural decisions, including Goroutine channel synchronization, MongoDB atomic `$inc` updates, and Redis Pub/Sub lifecycle management."*
5. **Conclusion & Live URL (4:00 - 4:30)**:
   - Display the live deployed link and GitHub repository.

---

## Technical Interview Preparation (Q&A)

Prepare for your technical interview with these detailed questions on the stack:

### Q1: Why did you use Redis Pub/Sub instead of having Go communicate directly with WebSockets?
> **Answer**: *"If the application is scaled across multiple server instances or containers behind a load balancer, Browser A might be connected to Go Server 1 while Browser B submits a vote to Go Server 2. Without Redis, Server 2 would have no way of notifying the WebSocket client on Server 1. By publishing the vote event to a Redis channel (`poll:{code}`), all Go server instances subscribed to that channel receive the notification, ensuring real-time synchronization across a distributed cluster."*

### Q2: How does MongoDB prevent race conditions when two users vote at the exact same millisecond?
> **Answer**: *"Instead of reading the document, incrementing the count in Go memory, and writing it back (which causes write skew and lost updates), I used MongoDB's native `$inc` operator inside `FindOneAndUpdate`. MongoDB executes field-level increments atomically at the document storage engine layer. Even under high concurrency, every vote is counted accurately without locks or race conditions."*

### Q3: How does the Gorilla WebSocket implementation handle client disconnects?
> **Answer**: *"Each WebSocket client runs two dedicated goroutines: a `readPump` and a `writePump`. The `writePump` sends periodic ping frames every 54 seconds. The `readPump` expects a pong frame within 60 seconds. If a client closes their tab or drops network connectivity, the read deadline expires, triggering deferred cleanup: the client is unregistered from the Hub, its channel is closed, and if no viewers remain in the poll room, the Redis subscription is gracefully canceled."*

### Q4: How is JWT authentication verified in the Go backend?
> **Answer**: *"The `AuthMiddleware` inspects the HTTP `Authorization` header for `Bearer <token>`. It parses the token using `github.com/golang-jwt/jwt/v5`, verifies the HMAC-SHA256 digital signature against `JWT_SECRET`, checks expiration (`exp`), and injects the extracted user's MongoDB `ObjectID` and email into the Gin request context (`c.Set("userID", ...)`)."*

---

## Submission Checklist
- [x] Full flow tested end-to-end (Create Poll ➔ Share Link ➔ Vote ➔ Live Results).
- [x] Redis and MongoDB doing meaningful work (Pub/Sub + Atomic Persistence).
- [x] Clean separation of concerns (`/frontend` and `/backend`).
- [x] Linked MongoDB Manual Dashboard implemented (`/admin/db`).
- [x] Postman collection provided (`LivePoll.postman_collection.json`).
- [x] Docker Compose provided (`docker-compose.yml`).
- [x] Instructions for mandatory 3–5 min video prepared.
- [x] Email sent to `devhiring@hclguvi.com`.

**All the best with your internship evaluation!**
