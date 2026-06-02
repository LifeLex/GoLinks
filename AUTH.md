# Authentication & the session cookie

How GoLinks authenticates requests: email + password with **server-side sessions**. A login mints a random token; the browser holds it in an `HttpOnly` cookie; the database stores only the token's SHA-256 hash. No JWT, no signing secret.

This document focuses on the **session cookie lifecycle**. For the higher-level layering and the full endpoint access matrix, see [`ARCHITECTURE.md`](./ARCHITECTURE.md) (§ *Authentication & authorization*); for conventions, see [`CLAUDE.md`](./CLAUDE.md).

---

## Creation flow (login)

`POST /auth/setup` (first-run admin) is identical, except it routes through `Bootstrap` instead of `Login`.

```
 BROWSER                          GO BACKEND
 ───────                          ──────────
                                  ┌─ internal/handlers/auth.go ─────────────────────────┐
 POST /auth/login   ───────────▶ │ Login()                                  :100        │
 {email, password}               │   decodeJSON → LoginRequest                          │
                                  │   user, token, err := h.auth.Login(ctx, req)  :105  │
                                  └───────────────────────────┬─────────────────────────┘
                                                               │
                                  ┌─ internal/service/auth.go ─▼─────────────────────────┐
                                  │ Login()                                  :107        │
                                  │   email = normalizeEmail(...)                        │
                                  │   user  = users.GetByEmail(ctx, email) ──┐           │
                                  │            (repository/user.go)          │           │
                                  │   bcrypt.CompareHashAndPassword(...)     :118        │
                                  │            ↑ wrong/unknown → ErrInvalidCredentials   │
                                  │   token = openSession(ctx, user.ID) ─────┐  :130     │
                                  └──────────────────────────────────────────┼──────────┘
                                                                              │
                                  ┌─ openSession()  service/auth.go :241 ─────▼──────────┐
                                  │   raw, hash, _ = generateToken()         :243        │
                                  │     ├ rand.Read(32 bytes)   crypto/rand  :273        │
                                  │     ├ raw  = base64url(bytes)                        │
                                  │     └ hash = hex(sha256(raw))   hashToken :281        │
                                  │   session = {TokenHash: hash, UserID, ExpiresAt}     │
                                  │   sessions.Create(ctx, session) ─────────┐           │
                                  └──────────────────────────────────────────┼──────────┘
                                                                              │
                                  ┌─ repository/session.go :28 ───────────────▼──────────┐
                                  │   INSERT INTO sessions (token_hash, ...)             │
                                  │   ── stores the HASH only, never the raw token ──    │
                                  └──────────────────────────────────────────────────────┘
                                                               │ returns RAW token
                                  ┌─ back in handlers/auth.go ─▼─────────────────────────┐
                                  │ Login()                                              │
                                  │   h.setSessionCookie(w, token)           :110        │
                                  │     http.SetCookie(w, &http.Cookie{      :184–195    │
                                  │        Name: "golinks_session"  (:19)               │
                                  │        Value: <raw token>                            │
                                  │        HttpOnly: true                                │
                                  │        SameSite: Lax                                 │
                                  │        Secure: cfg.Auth.CookieSecure                 │
                                  │        MaxAge: SESSION_TTL_HOURS*3600 })             │
                                  └───────────────────────────┬─────────────────────────┘
        Set-Cookie: golinks_session=… ◀────────────────────── │
   (browser stores it; HttpOnly = JS can't read it)
```

**Key invariant:** the **raw** token exists only in transit and in the browser cookie. The database stores only `sha256(raw)`, so a database read cannot mint a valid cookie. A fresh token is minted on every login (no session fixation).

---

## Read-back flow (every subsequent request)

```
 BROWSER                              GO BACKEND
 Cookie: golinks_session=<raw>  ───▶  internal/handlers/middleware.go
                                      Authenticate()                          :36
                                        cookie = r.Cookie("golinks_session")  :38
                                        user = auth.Authenticate(ctx, raw) ───┐
                                                                              │
                                      internal/service/auth.go                │
                                        Authenticate()                  :125  ▼
                                          hash = hashToken(raw)
                                          session = sessions.GetByTokenHash(hash)
                                          if expired → delete row, return nil (anonymous)
                                          user = users.GetByID(session.UserID)
                                        ◀──────────────────────────────────────┘
                                      r = r.WithContext(WithUser(ctx, user))  :43
                                        ↑ no/invalid cookie → stays anonymous, never 401s
                                              │
                                              ▼
                                      RequireAuth / RequireAdmin subrouters gate writes
                                      (wired in cmd/server/main.go)
```

The global `Authenticate` middleware never rejects — an absent or invalid cookie simply means the request is anonymous, which is what keeps public reads (`/query/*`, `GET /api/links`, …) working. Rejection happens only on the `RequireAuth` / `RequireAdmin` subrouters that guard writes.

---

## The cookie

| Attribute  | Value | Why |
| ---------- | ----- | --- |
| `Name`     | `golinks_session` | defined once in `internal/handlers/auth.go:19` |
| `Value`    | raw 32-byte token, base64url-encoded | the secret; its hash is what's stored server-side |
| `HttpOnly` | `true` | JavaScript can't read it — mitigates XSS token theft |
| `SameSite` | `Lax` | blocks cross-site POST cookies — the main CSRF defense |
| `Secure`   | `cfg.Auth.CookieSecure` | HTTPS-only in production; off in dev so `http://localhost` works |
| `MaxAge`   | `SESSION_TTL_HOURS * 3600` | default 720h (30 days); expired sessions are also purged server-side hourly |

---

## Files involved, by responsibility

| File | Role in the cookie flow |
| ---- | ----------------------- |
| `internal/handlers/auth.go` | `Login`/`Setup` handlers; `setSessionCookie` (`:184`) builds the `http.Cookie`; cookie-name const (`:19`) |
| `internal/service/auth.go` | `Login`/`Bootstrap` (verify), `openSession` (`:241`), `generateToken` (`:271`), `hashToken` (`:281`) |
| `internal/repository/session.go` | `Create` (`:28`) inserts the hashed token + expiry; `GetByTokenHash`, `DeleteExpired` |
| `internal/repository/user.go` | `GetByEmail` / `GetByID` for the credential check and read-back |
| `internal/handlers/middleware.go` | `Authenticate` (`:36`) reads the cookie back and loads the user into request context |
| `internal/config/config.go` | `AuthConfig` supplies `CookieSecure`, `SessionTTLHours`, `BcryptCost` |
| `cmd/server/main.go` | wires the global `Authenticate` middleware + the `RequireAuth` / `RequireAdmin` subrouters, and an hourly expired-session purge |

> Line numbers reflect the code at the time of writing and may drift; treat them as signposts, not contracts.
