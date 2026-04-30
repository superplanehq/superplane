from sqlalchemy import URL, create_engine
from sqlalchemy.engine import Engine
from sqlalchemy.orm import Session, sessionmaker


def build_engine(
    *,
    host: str,
    port: int,
    dbname: str,
    user: str,
    password: str,
    sslmode: str,
    application_name: str,
) -> Engine:
    url = URL.create(
        "postgresql+psycopg",
        username=user,
        password=password,
        host=host,
        port=port,
        database=dbname,
        query={"sslmode": sslmode, "application_name": application_name},
    )
    return create_engine(url, pool_pre_ping=True)


def build_session_factory(engine: Engine) -> sessionmaker[Session]:
    # expire_on_commit=False keeps ORM attributes accessible after commit so
    # callers can convert them to dataclasses without triggering lazy loads.
    return sessionmaker(bind=engine, expire_on_commit=False)
