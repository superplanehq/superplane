from pydantic import BaseModel, ConfigDict, Field


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
    """Suggest request body. LLM model comes from server env only, not from the client."""

    model_config = ConfigDict(extra="forbid")

    canvas_id: str = Field(min_length=1)
    node_id: str = Field(min_length=1)
    instruction: str = Field(min_length=1, max_length=2000)
    field_context_json: str = Field(default="", max_length=100 * 1024)
