from types import SimpleNamespace

from ai.tools.list_integration_resources import ListIntegrationResources


def test_list_integration_resources_requires_integration_id() -> None:
    ctx = SimpleNamespace(deps=SimpleNamespace(client=SimpleNamespace()))

    result = ListIntegrationResources.run(ctx, "", "repository")

    assert result == [
        {
            "__tool_error__": "integration_id is required",
            "__tool_name__": "list_integration_resources",
            "__tool_error_code__": "missing_integration_id",
        }
    ]


def test_list_integration_resources_requires_resource_type() -> None:
    ctx = SimpleNamespace(deps=SimpleNamespace(client=SimpleNamespace()))

    result = ListIntegrationResources.run(ctx, "integration-123", "")

    assert result == [
        {
            "__tool_error__": "type is required",
            "__tool_name__": "list_integration_resources",
            "__tool_error_code__": "missing_resource_type",
        }
    ]


def test_list_integration_resources_marks_empty_results() -> None:
    client = SimpleNamespace(list_integration_resources=lambda **kwargs: [])
    ctx = SimpleNamespace(deps=SimpleNamespace(client=client))

    result = ListIntegrationResources.run(ctx, "integration-123", "repository")

    assert result == [
        {
            "__tool_empty__": True,
            "__tool_name__": "list_integration_resources",
            "message": "No resources found",
            "integration_id": "integration-123",
            "resource_type": "repository",
        }
    ]
