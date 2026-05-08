from datetime import datetime, timezone

from src.auth.password_reset import issue_reset_token


def test_token_expires():
    issued = issue_reset_token("user@example.com", datetime(2026, 1, 1, tzinfo=timezone.utc))

    assert issued["expires_at"] == "2026-01-01T00:30:00+00:00"


def test_unknown_email_receives_same_response_shape():
    issued = issue_reset_token("missing@example.com", datetime(2026, 1, 1, tzinfo=timezone.utc))

    assert set(issued) == {"message", "token_hash", "expires_at"}
    assert issued["message"] == "If an account exists, reset instructions have been sent."
