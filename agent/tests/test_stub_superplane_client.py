from ai.evals.stub_superplane_client import StubSuperplaneClient


def test_stub_describe_canvas_is_empty() -> None:
    client = StubSuperplaneClient()
    summary = client.describe_canvas("c1")
    assert summary.canvas_id == "c1"
    assert summary.nodes == []
    assert summary.edges == []


def test_stub_lists_start_and_noop_unfiltered() -> None:
    client = StubSuperplaneClient()
    triggers = client.list_triggers()
    components = client.list_components()
    assert len(triggers) == 1
    assert triggers[0]["name"] == "start"
    assert len(components) == 1
    assert components[0]["name"] == "noop"


def test_stub_describe_by_name() -> None:
    client = StubSuperplaneClient()
    assert client.describe_trigger("start")["label"] == "Manual Run"
    assert client.describe_component("noop")["label"] == "No Operation"


def test_stub_list_org_integrations_empty() -> None:
    client = StubSuperplaneClient()
    assert client.list_org_integrations() == []
    assert client.list_integration_resources("any", "t") == []
