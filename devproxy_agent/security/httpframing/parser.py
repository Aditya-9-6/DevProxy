import re
from typing import Dict, List
from devproxy_agent.security.httpframing.verdict import Verdict, FramingResult


class HTTPFramingParser:
    _FIELD_NAME_REGEX = re.compile(r"^[!#$%&'*+\-.^_`|~0-9a-zA-Z]+$")

    @staticmethod
    def parse_headers_and_frame(header_bytes: bytes) -> FramingResult:
        try:
            raw_text = header_bytes.decode("iso-8859-1")
        except Exception:
            return FramingResult(Verdict.REJECT, "invalid_encoding")

        lines = raw_text.split("\r\n")
        if len(lines) < 1:
            return FramingResult(Verdict.REJECT, "empty_request")

        # Check for obsolete line folding (obs-fold: starting with SP or HTAB)
        for line in lines[1:]:
            if line and line[0] in (" ", "\t"):
                return FramingResult(Verdict.REJECT, "obsolete_line_folding")

        headers: Dict[str, List[str]] = {}
        for line in lines[1:]:
            if not line:
                continue
            if "\n" in line or "\r" in line:
                return FramingResult(Verdict.REJECT, "bare_cr_lf")
            parts = line.split(":", 1)
            if len(parts) != 2:
                return FramingResult(Verdict.REJECT, "malformed_header_line")
            field_name = parts[0].strip()
            field_val = parts[1].strip()

            if not HTTPFramingParser._FIELD_NAME_REGEX.match(field_name):
                return FramingResult(Verdict.REJECT, "invalid_field_name")

            field_lower = field_name.lower()
            if field_lower not in headers:
                headers[field_lower] = []
            headers[field_lower].append(field_val)

        # Validate Host header presence and uniqueness
        host_values = headers.get("host", [])
        if len(host_values) == 0:
            return FramingResult(Verdict.REJECT, "missing_host_header")
        if len(host_values) > 1:
            return FramingResult(Verdict.REJECT, "multiple_host_headers")

        # Validate Host value syntax
        host_val = host_values[0]
        if not host_val or any(c in host_val for c in (" ", "\t", "\r", "\n", "/")):
            return FramingResult(Verdict.REJECT, "invalid_host_header")

        # Content-Length multiplicity checks
        cl_values = headers.get("content-length", [])
        parsed_cl = None
        if len(cl_values) > 0:
            first_cl = cl_values[0]
            if not first_cl.isdigit():
                return FramingResult(Verdict.REJECT, "invalid_content_length")
            for cl in cl_values:
                if cl != first_cl:
                    return FramingResult(
                        Verdict.REJECT, "conflicting_content_length_multiplicity"
                    )
            try:
                parsed_cl = int(first_cl)
            except ValueError:
                return FramingResult(Verdict.REJECT, "invalid_content_length")

        # Transfer-Encoding checks
        te_values = headers.get("transfer-encoding", [])
        has_te = len(te_values) > 0
        is_chunked = False
        if has_te:
            # If multiple TE headers or chunked specified multiple times
            chunked_found = False
            for te in te_values:
                tokens = [t.strip().lower() for t in te.split(",")]
                for tok in tokens:
                    if tok == "chunked":
                        chunked_found = True
                    elif tok:
                        # Unknown or unsupported transfer encoding
                        pass
            is_chunked = chunked_found

        # Dual-interpretation smuggling checks (CL.TE vs TE.CL)
        if parsed_cl is not None and has_te:
            return FramingResult(
                Verdict.AMBIGUOUS, "cl_te_ambiguity", parsed_cl, has_te, is_chunked
            )

        if has_te and not is_chunked:
            return FramingResult(
                Verdict.AMBIGUOUS,
                "unsupported_transfer_encoding",
                parsed_cl,
                has_te,
                is_chunked,
            )

        return FramingResult(
            verdict=Verdict.ALLOW,
            reason="ok",
            content_length=parsed_cl,
            has_transfer_encoding=has_te,
            is_chunked=is_chunked,
        )
