from pathlib import Path


def test_runs_inside_container() -> None:
    assert Path("/.dockerenv").exists()
