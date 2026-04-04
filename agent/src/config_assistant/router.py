from fastapi import APIRouter, HTTPException, Request

from ai.jwt import JwtValidator
from config_assistant.agent import (
    build_config_assistant_agent,
    build_user_prompt,
    default_config_assistant_model,
)
from config_assistant.models import FieldSuggestOutput, SuggestHTTPRequest


def build_config_assistant_router() -> APIRouter:
    router = APIRouter()

    @router.post("/config-assistant/suggest")
    async def suggest_configuration_field(
        payload: SuggestHTTPRequest,
        request: Request,
    ) -> FieldSuggestOutput:
        from ai.repl_web import _resolve_required_bearer_token  # noqa: PLC0415

        token = _resolve_required_bearer_token(request)
        validator = JwtValidator.from_env()
        try:
            claims = validator.decode(token)
        except ValueError as exc:
            raise HTTPException(status_code=401, detail=str(exc)) from exc

        if claims.purpose != "config-assistant":
            raise HTTPException(status_code=403, detail="Invalid token purpose.")

        try:
            canvas_id = validator.validate_canvas(payload.canvas_id, claims)
        except ValueError as exc:
            raise HTTPException(status_code=403, detail=str(exc)) from exc

        if canvas_id != payload.canvas_id.strip():
            raise HTTPException(status_code=400, detail="canvas_id mismatch.")

        raw_model = (payload.model or "").strip()
        model_name = raw_model if raw_model else default_config_assistant_model()

        user_prompt = build_user_prompt(
            instruction=payload.instruction,
            field_context_json=payload.field_context_json,
            node_id=payload.node_id.strip(),
        )

        agent = build_config_assistant_agent(model=model_name)
        try:
            result = await agent.run(user_prompt=user_prompt)
        except Exception as exc:
            raise HTTPException(status_code=500, detail="Config assistant run failed.") from exc

        output = result.output
        if not isinstance(output, FieldSuggestOutput):
            raise HTTPException(status_code=500, detail="Unexpected assistant output.")

        return output

    return router
