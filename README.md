# Auth & User Service Prototype

This repository is extracted from my **Transcendence** project at [Hive Helsinki](https://www.hive.fi/).

During the Transcendence project, our team built an online Ping-Pong game with a microservice architecture.  
I was responsible for designing and implementing the **auth/user service** with **Go** and **Gin**.

I also implemented a minimal **frontend prototype** using **Svelte** for learning purposes.

---

## Demo

- Frontend: [https://auth-demo-sage.vercel.app/](https://auth-demo-sage.vercel.app/)
- Backend Swagger: [https://auth-demo-x0sd.onrender.com/api/docs/index.html](https://auth-demo-x0sd.onrender.com/api/docs/index.html)

## Features

Currently supported features include:

- User registration
- Login with username or email
- Logout
- Avatar update
- OAuth login (Google)
- Two-factor authentication (TOTP)
- Friends system
  - Friend listing
  - Friend requests
  - Online status tracking

## Libraries

### Backend

- `gin`: web framework
- `gorm`: ORM
- `go-playground/validator v10`: data validation
- `godotenv`: environment variables
- `slog-gin`: logging
- `gin-swagger`: Swagger (OpenAPI) docs

### Frontend

- `Svelte`: frontend framework
- `Tailwind CSS` : CSS
- `shadcn/ui (Svelte)`: UI library
- `Zod`: Validator
- `SvelteKit Superforms`: Form (SPA)

---

## Running the Project

Please make sure you have [Go](https://go.dev/) installed.

### Clone the repository:

```bash
git clone https://github.com/danielxfeng/auth-user-prototype.git
cd auth-user-prototype
```

### Backend

```bash
cd backend
make dev
```

Then navigate to `http://localhost:3003/api/docs/index.html` for swagger.

### Frontend

```bash
cd frontend
pnpm run dev
```

Then navigate to `http://localhost:5173`.
Note: Google login does not work locally until Google OAuth credentials are configured.

## Limitations

Due to the constraints of the Hive project, `SQLite` was required for the project.

As a result:

- `SQLite` is used to store authentication tokens and heartbeat data. In production, these would be better handled by `Redis`.
- Stale tokens and heartbeat data are not automatically cleaned up, and token auto-renewal is not implemented.
- On the frontend side, friend auto-completion is implemented in a basic manner.
