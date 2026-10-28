import os
from sqlmodel import SQLModel, create_engine, Session
from devproxy_agent.config import settings

# Resolve database URL with fallback to Settings (which provides defaults) or use SQLite for dev
DATABASE_URL = os.getenv("DATABASE_URL", settings.DATABASE_URI)

engine = create_engine(DATABASE_URL, echo=False)


def get_session() -> Session:
    """Yield a SQLModel session."""
    with Session(engine) as session:
        yield session


def init_db():
    """Create all tables defined in the models module."""
    SQLModel.metadata.create_all(engine)
