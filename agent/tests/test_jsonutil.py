from pydantic_ai.usage import RunUsage

from ai.jsonutil import to_jsonable


def test_to_jsonable_run_usage_roundtrip_shape() -> None:
    usage = RunUsage(
        requests=2,
        tool_calls=5,
        input_tokens=100,
        output_tokens=50,
        details={"some_provider_metric": 3},
    )
    payload = to_jsonable(usage)
    assert isinstance(payload, dict)
    assert payload["requests"] == 2
    assert payload["tool_calls"] == 5
    assert payload["input_tokens"] == 100
    assert payload["output_tokens"] == 50
    assert payload["details"] == {"some_provider_metric": 3}


def test_to_jsonable_empty_run_usage() -> None:
    payload = to_jsonable(RunUsage())
    assert payload["input_tokens"] == 0
    assert payload["output_tokens"] == 0
    assert payload["requests"] == 0
