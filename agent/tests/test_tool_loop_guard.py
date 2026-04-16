from ai.tool_loop_guard import ToolLoopGuard, build_tool_signature, is_no_progress_tool_result


def test_build_tool_signature_normalizes_args_shape() -> None:
    first = build_tool_signature(
        "list_integration_resources",
        {
            "type": " repository ",
            "integration_id": "abc",
            "parameters": {"page": "1", "query": " hello  world "},
        },
    )
    second = build_tool_signature(
        "list_integration_resources",
        {
            "parameters": {"query": "hello world", "page": "1"},
            "integration_id": "abc",
            "type": "repository",
        },
    )

    assert first == second


def test_no_progress_tool_result_detects_empty_and_structured_failures() -> None:
    assert is_no_progress_tool_result([])
    assert is_no_progress_tool_result({"__tool_error__": "boom"})
    assert is_no_progress_tool_result([{"__tool_empty__": True, "message": "No resources found"}])
    assert is_no_progress_tool_result([{"__tool_error__": "boom"}])
    assert not is_no_progress_tool_result([{"name": "repo-a"}])


def test_tool_loop_guard_triggers_after_repeated_no_progress_results() -> None:
    guard = ToolLoopGuard(max_repeated_no_progress=3)
    args = {"integration_id": "abc", "type": "repository"}

    for index in range(2):
        call_id = f"call-{index}"
        guard.register_call(call_id, "list_integration_resources", args)
        assert (
            guard.observe_result(
                call_id,
                "list_integration_resources",
                [{"__tool_empty__": True, "message": "No resources found"}],
            )
            is None
        )

    guard.register_call("call-3", "list_integration_resources", args)
    decision = guard.observe_result(
        "call-3",
        "list_integration_resources",
        [{"__tool_empty__": True, "message": "No resources found"}],
    )

    assert decision is not None
    assert decision.tool_name == "list_integration_resources"
    assert decision.repeated_count == 3
    assert "repeating the same discovery step" in decision.message.lower()


def test_tool_loop_guard_resets_after_progress() -> None:
    guard = ToolLoopGuard(max_repeated_no_progress=2)
    args = {"integration_id": "abc", "type": "repository"}

    guard.register_call("call-1", "list_integration_resources", args)
    assert (
        guard.observe_result(
            "call-1",
            "list_integration_resources",
            [{"__tool_error__": "integration_id is required"}],
        )
        is None
    )

    guard.register_call("call-2", "list_integration_resources", args)
    assert (
        guard.observe_result("call-2", "list_integration_resources", [{"name": "repo-a"}]) is None
    )
    assert guard.consecutive_no_progress_count == 0
