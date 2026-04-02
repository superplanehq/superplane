import os


class Config:
    def __init__(self) -> None:
        self.ai_model: str = self._parse_str("AI_MODEL", default="test")
        self.debug: bool = self._parse_bool("REPL_WEB_DEBUG")
        self.cors_origins: str = self._parse_str("REPL_WEB_CORS_ORIGINS", default="*")

        self.superplane_base_url: str = self._parse_str("SUPERPLANE_BASE_URL")
        self.superplane_user_agent: str = self._parse_str("SUPERPLANE_USER_AGENT", default="curl/8.7.1")

        self.drain_timeout: float = self._parse_float(
            "DRAIN_TIMEOUT", lower=0, upper=1000, default=300.0,
        )

        self.db_host: str = self._parse_str("DB_HOST", default="db")
        self.db_port: int = self._parse_int("DB_PORT", lower=1, upper=65535, default=5432)
        self.db_name: str = self._parse_str("DB_NAME")
        self.db_username: str = self._parse_str("DB_USERNAME")
        self.db_password: str = self._parse_str("DB_PASSWORD")
        self.db_sslmode: str = self._parse_str("DB_SSLMODE", default="disable")
        self.application_name: str = self._parse_str("APPLICATION_NAME", default="superplane-agent")

        self.jwt_secret: str = self._parse_str("JWT_SECRET")

        self.grpc_host: str = self._parse_str("INTERNAL_GRPC_HOST", default="0.0.0.0")
        self.grpc_port: int = self._parse_int("INTERNAL_GRPC_PORT", lower=1, upper=65535, default=50061)

        self.pattern_dir: str = self._parse_str("AGENT_PATTERN_DIR")

    @staticmethod
    def _parse_float(env_name: str, *, lower: float, upper: float, default: float) -> float:
        raw = os.getenv(env_name, "").strip()
        if not raw:
            return default
        try:
            value = float(raw)
        except ValueError:
            return default
        return max(lower, min(value, upper))

    @staticmethod
    def _parse_int(env_name: str, *, lower: int, upper: int, default: int) -> int:
        raw = os.getenv(env_name, "").strip()
        if not raw:
            return default
        try:
            value = int(raw)
        except ValueError:
            return default
        return max(lower, min(value, upper))

    @staticmethod
    def _parse_bool(env_name: str, *, default: bool = False) -> bool:
        raw = os.getenv(env_name, "").strip().lower()
        if not raw:
            return default
        return raw in {"1", "true", "yes", "on"}

    @staticmethod
    def _parse_str(env_name: str, *, default: str = "") -> str:
        return os.getenv(env_name, "").strip() or default


config = Config()
