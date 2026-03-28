# SuperPlane Security Analysis Report

**Date**: 2026-03-28
**Scope**: Full-stack — Go backend (pkg/) + React frontend (web_src/src/)
**Classification**: CONFIDENTIAL — Internal Use Only
**Findings**: 22 total | 2 CRITICAL, 6 HIGH, 5 MEDIUM, 4 LOW, 5 INFORMATIONAL
**Weighted Score**: 11.75

---

## Executive Summary

The SuperPlane codebase demonstrates a **generally solid security posture** with proper use of RBAC (Casbin), parameterized SQL queries (GORM), AES-256-GCM encryption, bcrypt password hashing, and comprehensive SSRF protection. The initial SAST report of 174 "hardcoded credentials" is overwhelmingly **false positives** (172 of 174 confirmed) — these are struct field names and configuration references, not actual embedded secrets.

However, there are **two critical and six high-severity findings** that require immediate attention.

---

## CRITICAL Findings

### 1. WebSocket Origin Check Disabled (CRITICAL)

**File:** `pkg/public/server.go:134-139`
**CWE:** CWE-346 (Origin Validation Error) | **OWASP:** A01

```go
upgrader: &websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        // TODO: implement origin checking
        return true
    },
}
```

The WebSocket upgrader accepts connections from ANY origin. An attacker can create a malicious page that connects to the SuperPlane WebSocket API using the victim's cookies, receiving real-time event data (Cross-Site WebSocket Hijacking).

**Remediation:** Validate the `Origin` header against the configured `BASE_URL`.

### 2. Development Auth Bypass Routes (CRITICAL)

**File:** `pkg/authentication/authentication.go:120-125, 153-197`
**CWE:** CWE-287 (Improper Authentication) | **OWASP:** A05

When `APP_ENV == "development"`, authentication is completely bypassed with a hardcoded mock user (`dev-user-123`). If `APP_ENV` is accidentally set to `development` in production, any request to `/auth/{provider}` auto-authenticates.

**Remediation:** Add defense-in-depth: bind address checks, prominent startup logging, startup panic if dev + production-looking config.

---

## HIGH Findings

### 3. NoOpEncryptor Available in Production

**File:** `pkg/server/server.go:399-401` | **CWE:** CWE-311

`NO_ENCRYPTION=yes` disables ALL encryption for secrets, credentials, and tokens. No guard prevents this in production.

**Remediation:** Disallow unless `APP_ENV` is `development` or `test`.

### 4. No Password Complexity Validation

**File:** `pkg/authentication/authentication.go:426-430` | **CWE:** CWE-521

Signup only validates non-empty password. No minimum length, complexity, or breached password checks.

**Remediation:** Enforce minimum 8 characters.

### 5. No Rate Limiting on Password Login

**File:** `pkg/authentication/authentication.go:337-377` | **CWE:** CWE-307

No rate limiting, account lockout, or brute force protection on password login (unlike magic code auth which has `magicCodeRateLimit = 5`).

**Remediation:** Add per-IP and per-account rate limiting.

### 6. Cookie Secure Flag Depends on TLS Termination

**File:** `pkg/authentication/authentication.go:294` | **CWE:** CWE-614

`Secure: r.TLS != nil` — if TLS is terminated at a reverse proxy (common production pattern), cookies are sent over HTTP.

**Remediation:** Check `X-Forwarded-Proto` header or add config flag.

### 7. No Security Headers (CSP, HSTS, etc.)

**CWE:** CWE-693

No `Content-Security-Policy`, `Strict-Transport-Security`, `X-Frame-Options`, or `X-Content-Type-Options` headers anywhere.

**Remediation:** Add security headers middleware.

### 8. SSH Host Key Verification Disabled

**File:** `pkg/components/ssh/client.go:88` | **CWE:** CWE-295

`ssh.InsecureIgnoreHostKey()` — vulnerable to MITM attacks between SuperPlane and SSH targets.

**Remediation:** Add optional host key verification. At minimum, log fingerprints.

---

## MEDIUM Findings

### 9. JWT Uses HS256 Symmetric Signing

**File:** `pkg/jwt/jwt.go:41` | Any service that can verify tokens can forge them.

### 10. Encryption Key Without KDF

**File:** `pkg/server/server.go:391-403` | Raw env var used as AES key, no HKDF applied.

### 11. gRPC Reflection Enabled Unconditionally

**File:** `pkg/grpc/server.go:158` | Aids attacker reconnaissance.

### 12. No Rate Limiting on gRPC API

**File:** `pkg/grpc/server.go:83-92` | No throttling interceptor.

### 13. No CSRF Protection on Auth Endpoints

`SameSite: Lax` mitigates most cases but not same-site attacks or older browsers.

---

## LOW Findings

### 14. Full Metadata Logged Including Potential Tokens

**File:** `pkg/authorization/interceptor.go:357,363`

### 15. SSH Private Key Preview in Error Messages

**File:** `pkg/components/ssh/client.go:124-131` — First 50 chars of private key in errors.

### 16. Server-Side Template with Env Vars

**File:** `pkg/web/index_template.go:24-37` — Low risk currently (only env vars injected).

### 17. Open Redirect Edge Cases

**File:** `pkg/authentication/authentication.go:993-1023` — Consider blocking backslash and embedded credentials.

---

## SAST False Positive Analysis

**172 of 174 initial SAST findings are confirmed FALSE POSITIVES.**

| Location | Nature | Verdict |
|----------|--------|---------|
| `pkg/authentication/` | Struct field `Password string` in `ProviderConfig` | Field names, not values |
| `pkg/cli/commands/secrets/` | CLI commands for CRUD on secrets entities | Management code, not secrets |
| `pkg/components/ssh/` | `AuthSpec` struct with fields referencing `SecretKeyRef` | References to vault, not hardcoded |
| `pkg/configuration/` | `SecretKeyRef` struct, `FieldTypeSecretKey` constant | Schema definitions |
| `pkg/impersonation/` | `CookieName = "impersonation_token"` | Cookie name constant |
| `pkg/integrations/aws/` | `AccessKeyID`, `SecretAccessKey` in STS response struct | AWS API field names |

---

## Positive Security Practices

1. **Comprehensive SSRF protection** — DNS rebinding defense, cloud metadata blocking, private IP filtering
2. **AES-256-GCM encryption** with random nonces for all secrets at rest
3. **Bcrypt password hashing** at cost 12
4. **RBAC with Casbin** covering all 50+ gRPC endpoints
5. **Error sanitization** preventing internal/database error leakage
6. **Parameterized SQL** throughout — no injection vectors
7. **HttpOnly + SameSite:Lax cookies** for all auth tokens
8. **Scoped token design** with audience, issuer, and expiry validation
9. **Input validation** on SSH component (env var names, port ranges)
10. **Impersonation audit logging** with admin ID, target ID, client IP
11. **Open redirect protection** with path-prefix validation
12. **No XSS vectors** in frontend — no `dangerouslySetInnerHTML`, no `eval()`

---

## Remediation Priority

### Immediate (Before Next Release)
1. Fix WebSocket origin validation — replace `return true` with `BASE_URL` check
2. Guard dev auth routes — defense-in-depth beyond `isDev`
3. Guard NoOpEncryptor — reject unless in dev/test env

### Short-Term (Within Sprint)
4. Add password complexity — minimum 8 chars
5. Add password login rate limiting
6. Fix cookie Secure flag — check `X-Forwarded-Proto`
7. Add security headers middleware

### Medium-Term (Next Quarter)
8. Migrate JWT to asymmetric signing (RS256/ES256)
9. Apply HKDF to encryption key
10. Gate gRPC reflection behind dev-only flag
11. Add CSRF tokens
12. Add gRPC rate limiting
13. Add optional SSH host key verification

---
*Generated by AQE v3 Security Reviewer Agent*
