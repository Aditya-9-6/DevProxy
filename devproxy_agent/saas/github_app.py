import os
import jwt
import time
import requests

# Placeholder implementation for GitHub App token retrieval.
# In production, replace with real App ID and private key.

def _create_jwt(app_id: str, private_key: str) -> str:
    """Create a JWT for GitHub App authentication.
    The JWT is valid for 10 minutes.
    """
    now = int(time.time())
    payload = {
        "iat": now - 60,
        "exp": now + (10 * 60),
        "iss": app_id,
    }
    return jwt.encode(payload, private_key, algorithm="RS256")


def get_installation_access_token(installation_id: int) -> str:
    """Return a short‑lived installation access token (valid ~1 hour).
    Uses placeholder credentials; returns a dummy token if environment variables
    are not set.
    """
    app_id = os.getenv("GITHUB_APP_ID", "placeholder_app_id")
    private_key = os.getenv("GITHUB_APP_PRIVATE_KEY", "placeholder_private_key")
    # In a real implementation, the private_key would be the PEM contents.
    if app_id.startswith("placeholder") or private_key.startswith("placeholder"):
        # Return a dummy token for development/testing.
        return "ghp_dummy_token_for_testing"
    jwt_token = _create_jwt(app_id, private_key)
    headers = {"Authorization": f"Bearer {jwt_token}", "Accept": "application/vnd.github+json"}
    url = f"https://api.github.com/app/installations/{installation_id}/access_tokens"
    resp = requests.post(url, headers=headers)
    resp.raise_for_status()
    return resp.json()["token"]
