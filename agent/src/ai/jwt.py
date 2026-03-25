import os
from dataclasses import dataclass

import jwt


def _normalize_optional(value: str | None) -> str | None:
    if value is None:
        return None
    normalized = value.strip()
    return normalized or None


@dataclass(frozen=True)
class Permission:
    resource_type: str
    action: str
    resources: list[str]


@dataclass(frozen=True)
class JwtClaims:
    subject: str
    org_id: str
    purpose: str
    permissions: list[Permission]


class JwtValidator:
    def __init__(self, jwt_secret: str, audience: str = "superplane_api") -> None:
        self._jwt_secret = jwt_secret
        self._audience = audience

    @classmethod
    def from_env(cls) -> "JwtValidator":
        jwt_secret = _normalize_optional(os.getenv("JWT_SECRET"))
        if jwt_secret is None:
            raise ValueError("Missing required setting: JWT_SECRET")
        return cls(jwt_secret=jwt_secret)

    def decode(self, token: str) -> JwtClaims:
        try:
            payload = jwt.decode(
                token,
                self._jwt_secret,
                algorithms=["HS256"],
                audience=self._audience,
            )
        except jwt.ExpiredSignatureError as error:
            raise ValueError("JWT has expired.") from error
        except jwt.ImmatureSignatureError as error:
            raise ValueError("JWT is not active yet.") from error
        except jwt.InvalidAudienceError as error:
            raise ValueError("Invalid JWT audience.") from error
        except jwt.InvalidAlgorithmError as error:
            raise ValueError("Invalid JWT algorithm.") from error
        except jwt.InvalidTokenError as error:
            raise ValueError("Invalid JWT.") from error

        if not isinstance(payload, dict):
            raise ValueError("Invalid JWT payload.")

        token_type = payload.get("token_type")
        if token_type != "scoped":
            raise ValueError("Invalid JWT type.")

        purpose = payload.get("purpose")
        if not isinstance(purpose, str) or purpose.strip() != "agent_chat":
            raise ValueError("Invalid JWT purpose.")

        subject = payload.get("sub")
        if not isinstance(subject, str) or not subject.strip():
            raise ValueError("JWT subject is required.")

        org_id = payload.get("org_id")
        if not isinstance(org_id, str) or not org_id.strip():
            raise ValueError("JWT org_id is required.")

        permissions = payload.get("permissions")
        if not isinstance(permissions, list) or len(permissions) == 0:
            raise ValueError("JWT permissions are required.")

        scoped_permissions: list[Permission] = []
        for permission in permissions:
            if not isinstance(permission, dict):
                raise ValueError("Scoped token permissions are invalid.")

            resource_type = permission.get("resourceType")
            action = permission.get("action")
            raw_resources = permission.get("resources")

            if not isinstance(resource_type, str) or not resource_type.strip():
                raise ValueError("Scoped token permission resourceType is required.")
            if not isinstance(action, str) or not action.strip():
                raise ValueError("Scoped token permission action is required.")
            if raw_resources is None:
                resources: list[str] = []
            elif isinstance(raw_resources, list) and all(isinstance(resource, str) for resource in raw_resources):
                resources = raw_resources
            else:
                raise ValueError("Scoped token permission resources are invalid.")

            scoped_permissions.append(
                Permission(
                    resource_type=resource_type,
                    action=action,
                    resources=resources,
                )
            )

        if not scoped_permissions:
            raise ValueError("Scoped token permissions are required.")

        return JwtClaims(
            subject=subject.strip(),
            org_id=org_id.strip(),
            purpose=purpose.strip(),
            permissions=scoped_permissions,
        )

    def allowed_canvas_ids(self, claims: JwtClaims) -> list[str]:
        canvas_ids: list[str] = []
        for permission in claims.permissions:
            if permission.resource_type != "canvases" or permission.action != "read":
                continue
            for resource in permission.resources:
                if resource not in canvas_ids:
                    canvas_ids.append(resource)

        return canvas_ids

    def validate_canvas(
        self,
        requested_canvas_id: str | None,
        claims: JwtClaims,
    ) -> str:
        canvas_id = _normalize_optional(requested_canvas_id)
        if canvas_id is None:
            raise ValueError("Missing required request field: canvas_id")

        allowed_canvas_ids = self.allowed_canvas_ids(claims)
        if allowed_canvas_ids and canvas_id not in allowed_canvas_ids:
            raise ValueError("Scoped token does not allow the requested canvas.")

        return canvas_id
