import logging
from gossipmesh.security.tls_entropy import analyze_tls_client_hello

logger = logging.getLogger(__name__)


class ErrBotnetDetected(Exception):
    """Raised when a TLS handshake is rejected due to low entropy botnet fingerprint."""

    pass


class HandshakeInterceptor:
    """Intercepts TLS connections and applies entropy analysis in Python."""

    def __init__(self, threshold: int = 80) -> None:
        self.threshold = threshold

    async def intercept(self, client_hello: bytes) -> None:
        _, anomaly_score = analyze_tls_client_hello(client_hello)
        if anomaly_score >= self.threshold:
            raise ErrBotnetDetected(
                "proxy: tls handshake rejected due to low entropy botnet fingerprint"
            )
