from __future__ import annotations

from datetime import datetime, timedelta, timezone
from hashlib import sha256


TOKEN_TTL = timedelta(minutes=30)


def issue_reset_token(email: str, now: datetime | None = None) -> dict[str, str]:
    now = now or datetime.now(timezone.utc)
    token = sha256(f"{email}:{now.isoformat()}".encode()).hexdigest()
    expires_at = now + TOKEN_TTL
    return {
        "message": "If an account exists, reset instructions have been sent.",
        "token_hash": sha256(token.encode()).hexdigest(),
        "expires_at": expires_at.isoformat(),
    }
