from config_assistant.agent import build_config_assistant_agent, build_user_prompt, load_system_prompt


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
