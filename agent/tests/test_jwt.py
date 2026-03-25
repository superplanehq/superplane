import jwt

from ai.jwt import JwtValidator


def test_jwt_validator_decodes_agent_builder_token() -> None:
    token = jwt.encode(
        {
            "aud": "superplane_api",
            "token_type": "scoped",
            "purpose": "agent-builder",
            "sub": "user-123",
            "org_id": "org-123",
            "scopes": [
                "canvases:read:canvas-123",
                "canvases:read:canvas-123",
                "org:read",
            ],
        },
        "secret",
        algorithm="HS256",
    )

    claims = JwtValidator(jwt_secret="secret").decode(token)

    assert claims.subject == "user-123"
    assert claims.org_id == "org-123"
    assert claims.purpose == "agent-builder"
    assert claims.scopes == ["canvases:read:canvas-123", "org:read"]
