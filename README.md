# Auth & User Service Prototype

This repository is extracted from my **Transcendence** project at [Hive Helsinki](https://www.hive.fi/).

During the Transcendence project, our team built an online Ping-Pong game with a microservice architecture.  
I was responsible for designing and implementing the **auth/user service** with **Go** and **Gin**.

I also implemented a minimal **frontend prototype** using **Svelte** for learning purposes.
It's just a quick demo, so for UI/UX I just applied the default theme from [shadcnui](https://ui.shadcn.com/themes).

---

## Demo

- Frontend: [https://auth-demo-sage.vercel.app/](https://auth-demo-sage.vercel.app/)
- Backend Swagger: [https://auth-demo-x0sd.onrender.com/api/docs/index.html](https://auth-demo-x0sd.onrender.com/api/docs/index.html)



https://github.com/user-attachments/assets/550b8a53-5775-47e8-8d4f-c42a7e174bf7



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
- Redis-backed session tokens (Optional) (revocation + sliding expiration)
- Redis-backed heartbeats for online status (Optional)

## Libraries

### Backend

- `gin`: web framework
- `gorm`: ORM
- `go-redis`: Redis
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

Redis is optional. To enable it locally:

```bash
# example: run redis with docker
docker run --rm -p 6379:6379 redis:latest

# enable redis mode for the backend
export REDIS_URL=redis://localhost:6379/0
```

Token extension (sliding expiration) in Redis mode:

- `USER_TOKEN_EXPIRY` controls the Redis TTL and is extended on token validation.
- `USER_TOKEN_ABSOLUTE_EXPIRY` caps the maximum lifetime via the JWT `exp` claim.

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

- The project still uses `SQLite` for core data due to Hive constraints.
- Redis-backed tokens and heartbeats are implemented, but the sliding expiration and cleanup strategy is simple.
- On the frontend side, friend auto-completion is implemented in a basic manner.
