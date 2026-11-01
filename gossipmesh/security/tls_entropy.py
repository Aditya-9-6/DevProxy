import math
from typing import Tuple


def calculate_shannon_entropy(data: bytes) -> float:
    if not data:
        return 0.0
    freq = [0] * 256
    for b in data:
        freq[b] += 1
    length = float(len(data))
    entropy = 0.0
    for count in freq:
        if count == 0:
            continue
        p = float(count) / length
        entropy -= p * math.log2(p)
    return entropy


def analyze_tls_client_hello(client_hello: bytes) -> Tuple[float, int]:
    if len(client_hello) < 5:
        return 0.0, 100
    entropy = calculate_shannon_entropy(client_hello)
    if entropy < 2.5:
        anomaly_score = 90
    elif entropy < 4.0:
        anomaly_score = 50
    else:
        anomaly_score = 0
    return entropy, anomaly_score
