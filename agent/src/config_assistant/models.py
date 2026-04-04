from pydantic import BaseModel, Field


class FieldSuggestOutput(BaseModel):
    """Structured LLM output for a single configuration field."""

    value: str = Field(
        min_length=1,
        description="Suggested field value (expression or literal).",
    )
    explanation: str | None = Field(
        default=None,
        description="Short optional note for the user about the suggestion.",
    )


class SuggestHTTPRequest(BaseModel):
    canvas_id: str = Field(min_length=1)
    node_id: str = Field(min_length=1)
    instruction: str = Field(min_length=1, max_length=2000)
    field_context_json: str = Field(default="", max_length=100 * 1024)
    # When omitted, the handler uses CONFIG_ASSISTANT_AI_MODEL / AI_MODEL from the environment.
    model: str | None = Field(default=None, max_length=200)
