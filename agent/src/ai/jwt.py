import os
import jwt
from dataclasses import dataclass


from ai.text import normalize_optional


@dataclass(frozen=True)
class JwtClaims:
    subject: str
    org_id: str
    purpose: str
    scopes: list[str]

class JwtValidator:
    def __init__(self, jwt_secret: str, audience: str = "superplane_api") -> None:
        self._jwt_secret = jwt_secret
        self._audience = audience

    @classmethod
    def from_env(cls) -> "JwtValidator":
        jwt_secret = normalize_optional(os.getenv("JWT_SECRET"))
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

        if payload.get("token_type") != "scoped":
            raise ValueError("Invalid JWT type.")

        purpose = payload.get("purpose")
        if not isinstance(purpose, str) or purpose.strip() != "agent-builder":
            raise ValueError("purpose is required.")

        subject = payload.get("sub")
        if not isinstance(subject, str) or not subject.strip():
            raise ValueError("subject is required.")

        org_id = payload.get("org_id")
        if not isinstance(org_id, str) or not org_id.strip():
            raise ValueError("org_id is required.")

        scopes = payload.get("scopes")
        if not isinstance(scopes, list) or len(scopes) == 0:
            raise ValueError("JWT scopes are required.")

        return JwtClaims(
            subject=subject.strip(),
            org_id=org_id.strip(),
            purpose=purpose.strip(),
            scopes=self._parse_scopes(scopes),
        )

    def _parse_scopes(self, raw: list[object]) -> list[str]:
        scopes: list[str] = []
        for scope in raw:
            if not isinstance(scope, str):
                raise ValueError("Scopes are invalid.")
            normalized = scope.strip()
            if not normalized:
                continue
            if normalized not in scopes:
                scopes.append(normalized)

        if not scopes:
            raise ValueError("Scopes are required.")

        return scopes

    def allowed_canvas_ids(self, claims: JwtClaims) -> list[str]:
        canvas_ids: list[str] = []
        for scope in claims.scopes:
            parts = scope.split(":")
            if len(parts) != 3:
                continue
            resource_type, action, resource_id = parts
            if resource_type != "canvases" or action != "read":
                continue
            normalized_resource = resource_id.strip()
            if normalized_resource and normalized_resource not in canvas_ids:
                canvas_ids.append(normalized_resource)

        return canvas_ids

    def validate_canvas(
        self,
        requested_canvas_id: str | None,
        claims: JwtClaims,
    ) -> str:
        canvas_id = normalize_optional(requested_canvas_id)
        if canvas_id is None:
            raise ValueError("Missing required request field: canvas_id")

        allowed_canvas_ids = self.allowed_canvas_ids(claims)
        if not allowed_canvas_ids:
            raise ValueError("Scoped token does not allow canvases.")
        if canvas_id not in allowed_canvas_ids:
            raise ValueError("Scoped token does not allow the requested canvas.")

        return canvas_id
