# Security Policy

We take the security of this project seriously. This platform stores and
transfers files for organisations, so we treat vulnerabilities with priority.

## Supported versions

| Version        | Supported          |
| -------------- | ------------------ |
| `main`         | ✅ Yes             |
| Latest release | ✅ Yes             |
| Older releases | ⚠️ Best-effort     |

## Reporting a vulnerability

**Please do not open a public issue for security problems.**

Report privately via GitHub Security Advisories:

1. Go to the repository's **Security → Advisories → Report a vulnerability**.
2. Provide a clear description, affected version/commit, reproduction steps, and
   impact assessment.

If you cannot use GitHub Advisories, email **security@sapphirebroking.com** with
the same details. Encrypt sensitive details if possible.

### What to expect

- **Acknowledgement** within 3 business days.
- A **triage assessment** and severity rating (CVSS) within 7 business days.
- Coordinated disclosure: we will agree on a timeline and credit you (unless you
  prefer to remain anonymous).

## Scope

In scope:

- Authentication / authorization bypass (JWT, API keys, RBAC, SFTP auth)
- Path traversal, SSRF, injection (SQL, command, template)
- Encryption-at-rest weaknesses, insecure key handling
- Broken access control on files, shares, admin/backup endpoints
- Data exposure across tenants/users

Out of scope:

- Vulnerabilities requiring physical access to the host
- Denial of service from unrealistic traffic volumes
- Issues in third-party dependencies already tracked upstream (report to them,
  but tell us so we can pin/patch)

## Hardening built in

- Argon2id password hashing, HS256 JWT with short TTLs
- AES-256 encryption at rest (opt-in), streaming encrypted backups
- RBAC with least-privilege roles, per-file grants, DLP on sharing
- Immutable, compliance-grade audit trail + anomaly detection
- Strict security headers (`X-Content-Type-Options`, `X-Frame-Options`, etc.)

Thank you for helping keep the project and its users safe.
