from fastapi import FastAPI
from fastapi.testclient import TestClient

from config_assistant.agent import build_config_assistant_agent, build_user_prompt, load_system_prompt
from config_assistant.router import build_config_assistant_router


def test_load_config_assistant_system_prompt() -> None:
    text = load_system_prompt()
    assert "SuperPlane" in text
    assert "value" in text.lower()


def test_build_user_prompt_includes_instruction() -> None:
    prompt = build_user_prompt(
        instruction="make it true",
        field_context_json='{"fieldName":"x"}',
        node_id="node-1",
    )
    assert "make it true" in prompt
    assert "node-1" in prompt
    assert "fieldName" in prompt


def test_build_config_assistant_agent_test_model() -> None:
    agent = build_config_assistant_agent(model="test")
    assert agent is not None


def test_config_assistant_suggest_missing_authorization_returns_401() -> None:
    app = FastAPI()
    app.include_router(build_config_assistant_router())
    client = TestClient(app)
    response = client.post(
        "/config-assistant/suggest",
        json={
            "canvas_id": "550e8400-e29b-41d4-a716-446655440000",
            "node_id": "node-1",
            "instruction": "hello",
            "field_context_json": "{}",
        },
    )
    assert response.status_code == 401
    assert response.json()["detail"] == "Authorization header is required."
