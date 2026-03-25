from pathlib import Path

from ai.patterns import list_decision_patterns, load_decision_patterns, search_decision_patterns


def test_load_decision_patterns_reads_markdown_files(tmp_path: Path) -> None:
    pattern_file = tmp_path / "ephemeral-machines.md"
    pattern_file.write_text(
        "# Ephemeral Preview Machines\n\n"
        "Keywords: ephemeral, preview, pull request\n\n"
        "Create infra on PR open and tear down after timeout.",
        encoding="utf-8",
    )

    patterns = load_decision_patterns(pattern_dir=tmp_path)

    assert len(patterns) == 1
    assert patterns[0].id == "ephemeral-machines"
    assert patterns[0].title == "Ephemeral Preview Machines"
    assert "ephemeral" in patterns[0].keywords


def test_search_decision_patterns_returns_relevant_matches(tmp_path: Path) -> None:
    (tmp_path / "ephemeral-preview-machines.md").write_text(
        "# Ephemeral PR Preview Machines\n\n"
        "Keywords: ephemeral, preview, github, pull request\n\n"
        "On PR open provision infra and post a URL. Later tear it down.",
        encoding="utf-8",
    )
    (tmp_path / "daily-digest.md").write_text(
        "# Daily Digest Notifications\n\n"
        "Keywords: digest, schedule, summary\n\n"
        "Send a daily summary to Slack.",
        encoding="utf-8",
    )

    results = search_decision_patterns(
        query="Can you design ephemeral machines for pull requests?",
        limit=2,
        pattern_dir=tmp_path,
    )

    assert len(results) == 1
    assert results[0]["id"] == "ephemeral-preview-machines"
    assert isinstance(results[0]["score"], int)
    assert results[0]["score"] > 0


def test_list_decision_patterns_returns_pattern_metadata(tmp_path: Path) -> None:
    (tmp_path / "foo.md").write_text("# Foo Pattern\n\nBody", encoding="utf-8")

    items = list_decision_patterns(pattern_dir=tmp_path)

    assert len(items) == 1
    assert items[0]["id"] == "foo"
    assert items[0]["title"] == "Foo Pattern"
