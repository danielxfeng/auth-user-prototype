# Auth & User Service Prototype

This repository is extracted from my **Transcendence** project at [Hive Helsinki](https://www.hive.fi/).

During the Transcendence project, our team built an online PingPong game with a microservice architecture.  
I was responsible for designing and implementing the **auth/user service** with **Go** and **Gin**.

I also implemented a minimal **frontend prototype** using **Svelte** for learning propose here.

---

## Features

Currently supported features include:

- User registration
- Login with username or email
- OAuth login (Google)
- Two-factor authentication (TOTP)
- Friends system
  - Friend listing
  - Friend requests
  - Online status tracking

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

### Frontend

```bash
#TODO
```


## Limitation
Due to the limitation of the Hive subject, `SQLite` was required for the project.

As a result:
 - `SQLite` is used to store `authentication tokens` and `heartbeat` data. In production, these would be better handled by `Redis`.
 - And also, `Stale token` and `heartbeat` data are not automatically cleaned up. And `Token` auto-renewal is not implemented.
 - On the frontend side, `Friend auto-completion` is implemented in a very rough way.