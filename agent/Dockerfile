FROM python:3.13-slim AS dev

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

COPY --from=ghcr.io/astral-sh/uv:0.6.6 /uv /uvx /bin/

WORKDIR /app/agent

RUN apt-get update \
    && apt-get install -y --no-install-recommends make bash \
    && rm -rf /var/lib/apt/lists/*

CMD ["bash", "-lc", "sleep infinity"]
