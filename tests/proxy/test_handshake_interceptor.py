import pytest
from gossipmesh.proxy.handshake_interceptor import (
    HandshakeInterceptor,
    ErrBotnetDetected,
)


@pytest.mark.asyncio
async def test_handshake_interceptor() -> None:
    interceptor = HandshakeInterceptor(threshold=80)

    malicious_hello = b"\x16\x03\x01\x00\x05\x01\x01\x01\x01\x01"
    with pytest.raises(ErrBotnetDetected):
        await interceptor.intercept(malicious_hello)

    normal_hello = b"\x16\x03\x01\x00\x20\x01\x0a\x3b\xff\x12\x44\x55\x66\x77\x88\x99"
    await interceptor.intercept(normal_hello)
