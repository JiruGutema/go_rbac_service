# Integration Guide

How applications in any stack consume this service. The pattern is identical everywhere; only the framework hook differs.

## Trust model

```
[IdP: Keycloak / Auth0 / Cognito]
        │  issues JWT
        ▼
[Your app or API gateway]  ── verifies JWT, extracts tenant_id + subject_id
        │
        │  Authorization: Bearer <per-app API key>   (+ mTLS in service meshes)
        ▼
[RBAC Service]  ── trusts the authenticated caller's subject assertion → ALLOW/DENY
```

- **The calling app (or its gateway) verifies the end-user JWT** — not this service. This is the industry-standard split (OpenFGA and SpiceDB work the same way).
- This service authenticates **applications**, not end users: each client app gets its own API key; rotate per app. mTLS is recommended transport hardening inside a mesh.
- Optional convenience mode (config-gated, off by default): the service verifies JWTs itself against a configured JWKS and extracts `tenant_id`/`subject_id` claims. Useful for edge middleware without an app in between.

**Enforcement always stays in your app**: this service decides; your code must actually block the request on deny. Treat network errors as deny (fail closed).

## FastAPI (Python)

Idiomatic hook: a `Depends` guard factory. (`current_user` is your existing dependency that verified the JWT.)

```python
import httpx
from fastapi import Depends, HTTPException

async def check(tenant: str, subject: str, resource: str, action: str) -> bool:
    r = await client.post(f"{RBAC_URL}/v1/check",           # client: shared httpx.AsyncClient
        json={"tenant_id": tenant, "subject_id": subject,
              "resource": resource, "action": action},
        headers={"Authorization": f"Bearer {RBAC_API_KEY}"})
    return r.status_code == 200 and r.json()["allowed"]

def require(resource: str, action: str):
    async def guard(user=Depends(current_user)):
        if not await check(user.tenant_id, user.sub, resource, action):
            raise HTTPException(status_code=403)
    return guard

@app.post("/invoices/approve", dependencies=[Depends(require("invoice", "approve"))])
async def approve_invoice(): ...
```

## ASP.NET Core (.NET)

Idiomatic hook: policy-based authorization — a requirement plus a handler that calls the service. Register the client as a singleton.

```csharp
public record PermissionRequirement(string Resource, string Action) : IAuthorizationRequirement;

public class RbacHandler(RbacClient rbac) : AuthorizationHandler<PermissionRequirement>
{
    protected override async Task HandleRequirementAsync(
        AuthorizationHandlerContext ctx, PermissionRequirement req)
    {
        var subject = ctx.User.FindFirstValue(ClaimTypes.NameIdentifier);
        var tenant  = ctx.User.FindFirstValue("tenant_id");
        var allowed = await rbac.CheckAsync(tenant, subject, req.Resource, req.Action);
        if (allowed) ctx.Succeed(req);
    }
}

// Program.cs
builder.Services.AddAuthorization(o =>
    o.AddPolicy("invoice:approve", p =>
        p.Requirements.Add(new PermissionRequirement("invoice", "approve"))));

// usage
[Authorize(Policy = "invoice:approve")]
public IActionResult Approve() { ... }
```

The subject comes from the already-validated `ClaimsPrincipal` (JWT bearer middleware) — the RBAC service never sees the raw token.

## Go

Idiomatic hooks: `net/http` middleware and gRPC unary interceptors, holding one long-lived gRPC connection to the service.

```go
func Require(rbac authzv1.AuthorizationServiceClient, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := identity.FromContext(r.Context()) // set by your JWT middleware
			resp, err := rbac.Check(r.Context(), &authzv1.CheckRequest{
				TenantId: id.Tenant, SubjectId: id.Subject,
				Resource: resource, Action: action,
			})
			if err != nil || !resp.Allowed { // fail closed
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

For gRPC servers, the same check goes in a `grpc.UnaryServerInterceptor` returning `codes.PermissionDenied`.

## Client/SDK guidance

- SDKs stay **thin**: connection management, API-key auth, `Check` / `BatchCheck` / `ListSubjectPermissions`. Nothing else.
- One long-lived gRPC channel (or pooled HTTP client) per process; enable keepalives. Never dial per request.
- **No client-side decision caching in v1** — the server's version-keyed cache is authoritative; client caches would reintroduce the stale-ALLOW problem after revocations.
- Use `BatchCheck` when guarding several actions at once and `ListSubjectPermissions` for menu/UI rendering — never a loop of `Check` calls.

## Deployment & latency

- Deploy the service in the same network/region as its consumers; every guarded request pays one round trip.
- Scale is horizontal: stateless nodes behind a load balancer, shared Postgres + Redis.
- Future options if a hop is ever too much: read-replica/sidecar mode and an embedded Go library mode are on the [roadmap](ROADMAP.md).
