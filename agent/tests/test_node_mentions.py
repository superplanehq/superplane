from ai.agent import AgentDeps
from ai.models import CanvasNode, CanvasSummary
from ai.node_mentions import expand_node_mentions_in_prompt, parse_node_mention_ids


def test_parse_node_mention_ids_dedupes_and_preserves_order() -> None:
    text = "Hello @[node:a] and @[node:b] again @[node:a]"
    assert parse_node_mention_ids(text) == ["a", "b"]


def test_expand_node_mentions_appends_appendix_and_primes_cache() -> None:
    class FakeClient:
        def describe_canvas(self, canvas_id: str) -> CanvasSummary:
            assert canvas_id == "c1"
            return CanvasSummary(
                canvas_id=canvas_id,
                nodes=[
                    CanvasNode(id="n1", name="Slack", type="action", block_name="slack.post"),
                    CanvasNode(id="n2", name="Other", type="trigger", block_name="http.webhook"),
                ],
                edges=[],
            )

    deps = AgentDeps(client=FakeClient(), canvas_id="c1")  # type: ignore[arg-type]
    q = "Fix @[node:n1]"
    out = expand_node_mentions_in_prompt(q, deps)

    assert q in out
    assert "Referenced nodes" in out
    assert "Slack" in out
    assert "`n1`" in out
    assert deps.canvas_cache["c1"] is not None
    assert len(deps.canvas_cache["c1"].nodes) == 2


def test_expand_node_mentions_unknown_id() -> None:
    class FakeClient:
        def describe_canvas(self, canvas_id: str) -> CanvasSummary:
            return CanvasSummary(canvas_id=canvas_id, nodes=[], edges=[])

    deps = AgentDeps(client=FakeClient(), canvas_id="c")  # type: ignore[arg-type]
    out = expand_node_mentions_in_prompt("See @[node:missing]", deps)
    assert "not found on canvas" in out
