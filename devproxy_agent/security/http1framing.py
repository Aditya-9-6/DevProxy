import re
from typing import Dict, List


class FramingAmbiguityError(ValueError):
    """Raised when an HTTP/1.1 request contains ambiguous framing headers."""

    pass


class HTTP1FramingDetector:
    """Validates RFC 9112 request framing and rejects smuggling vectors."""

    @staticmethod
    def validate_headers(headers: Dict[str, str]) -> List[str]:
        findings = []

        # Normalize header keys to lowercase for inspection
        norm_headers = {k.lower(): v for k, v in headers.items()}

        # 1. Check for multiple Content-Length headers with conflicting values
        content_length_values = []
        for k, v in headers.items():
            if k.lower() == "content-length":
                content_length_values.append(v.strip())

        if len(content_length_values) > 1:
            # Check if all values are identical
            if len(set(content_length_values)) > 1:
                findings.append("DUPLICATE_CONTENT_LENGTH_CONFLICT")
            else:
                findings.append("DUPLICATE_CONTENT_LENGTH")

        # 2. Check for both Transfer-Encoding and Content-Length
        has_transfer_encoding = "transfer-encoding" in norm_headers
        has_content_length = len(content_length_values) > 0

        if has_transfer_encoding and has_content_length:
            findings.append("TRANSFER_ENCODING_AND_CONTENT_LENGTH_CONFLICT")

        # 3. Validate Transfer-Encoding
        if has_transfer_encoding:
            te_val = norm_headers["transfer-encoding"].lower()
            if (
                "chunked" not in te_val
                and "compress" not in te_val
                and "deflate" not in te_val
                and "gzip" not in te_val
            ):
                findings.append("UNSUPPORTED_TRANSFER_CODING")

        # 4. Check for invalid Content-Length formatting
        for val in content_length_values:
            if not re.match(r"^\d+$", val):
                findings.append("INVALID_CONTENT_LENGTH_FORMAT")

        return findings

    @classmethod
    def assert_secure_framing(cls, headers: Dict[str, str]) -> None:
        findings = cls.validate_headers(headers)
        if findings:
            raise FramingAmbiguityError(
                f"HTTP/1.1 framing ambiguity detected: {', '.join(findings)}"
            )
