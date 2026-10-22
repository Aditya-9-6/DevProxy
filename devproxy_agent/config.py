import os
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    PROJECT_NAME: str = "DevProxy Agent"
    VERSION: str = "0.1.0"
    API_V1_STR: str = "/api/v1"

    # Core security keys – safe defaults for local development
    SECRET_KEY: str = os.getenv("SECRET_KEY", "dev-secret-key")
    ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30
    REFRESH_TOKEN_EXPIRE_DAYS: int = 7
    JWT_ISSUER: str = "devproxy_agent"
    JWT_AUDIENCE: str = "devproxy-api"

    ENCRYPTION_KEY: str = os.getenv("ENCRYPTION_KEY", "dev-encryption-key")

    # PostgreSQL connection – defaults point to the local dev database used by the SaaS stack
    POSTGRES_USER: str = os.getenv("POSTGRES_USER", "devuser")
    POSTGRES_PASSWORD: str = os.getenv("POSTGRES_PASSWORD", "devpass")
    POSTGRES_HOST: str = os.getenv("POSTGRES_HOST", "localhost")
    POSTGRES_PORT: str = os.getenv("POSTGRES_PORT", "5432")
    POSTGRES_DB: str = os.getenv("POSTGRES_DB", "devproxy_saas")

    REDIS_HOST: str = os.getenv("REDIS_HOST", "localhost")
    REDIS_PORT: str = os.getenv("REDIS_PORT", "6379")

    DEFAULT_INITIAL_CREDITS: int = int(os.getenv("DEFAULT_INITIAL_CREDITS", "1000"))
    ERROR_RATE_THRESHOLD: float = float(os.getenv("ERROR_RATE_THRESHOLD", "0.25"))
    PROVIDER_API_KEYS: list[str] = (
        os.getenv("PROVIDER_API_KEYS", "").split(",")
        if os.getenv("PROVIDER_API_KEYS")
        else []
    )
    GITHUB_CLIENT_SECRET: str | None = None

    OIDC_CLIENT_ID: str | None = None
    OIDC_CLIENT_SECRET: str | None = None
    OIDC_DISCOVERY_URL: str | None = None
    OIDC_REDIRECT_URI: str | None = None

    @property
    def DATABASE_URI(self) -> str:
        """Return full DATABASE_URL if set, otherwise use SQLite for local dev."""
        return os.getenv("DATABASE_URL") or "sqlite:///devproxy.db"

    @property
    def REDIS_URI(self) -> str:
        return (
            os.getenv("REDIS_URL") or f"redis://{self.REDIS_HOST}:{self.REDIS_PORT}/0"
        )

    # SaaS‑specific configuration
    SAAS_MODE: bool = True
    CYCLE_LOG_DB: str = "metrics.db"
    PR_PER_CYCLE_FACTOR: float = 0.5

    model_config = SettingsConfigDict(
        env_file=".env.devproxy_agent",
        env_file_encoding="utf-8",
        case_sensitive=True,
        extra="ignore",
    )


# Instantiate a single Settings object for the whole application
settings = Settings()  # type: ignore
