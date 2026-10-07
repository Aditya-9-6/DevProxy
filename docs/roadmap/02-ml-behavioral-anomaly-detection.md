# Feature Proposal: Local ML Behavioral Anomaly Detection (Shadow API & BOLA)

## 🎯 Objective
Static rules (like Aho-Corasick or Regex) are excellent for detecting known secrets, but they cannot catch sophisticated logic flaws such as Shadow APIs (undocumented endpoints), BOLA (Broken Object Level Authorization), or unusual payload mutations. We want to implement an embedded Machine Learning engine to baseline "normal" API traffic and flag behavioral anomalies in real-time.

## 💡 The Machine Learning Approach
DevProxy will embed a lightweight ML runtime to process traffic locally, ensuring zero data leaves the developer's machine, preserving privacy and latency.

### 1. ONNX Runtime Integration
- Integrate `github.com/yalue/onnxruntime_go` to run pre-trained or dynamically updated models directly inside the Analysis Worker pool.
- Use a lightweight, high-performance model architecture such as an **Isolation Forest** or an **Autoencoder**.

### 2. Feature Extraction Pipeline
- In the analysis path, extract mathematical features from the `TrafficEvent`:
  - Request/Response payload size vectors.
  - Header entropy (detecting unusual user-agent spoofing or unexpected tokens).
  - Parameter count and type variance (e.g., a parameter that is usually an Integer suddenly becomes a massive String).
  - Path structure sequencing (detecting brute-force or enumeration attempts).

### 3. Local Baselining & Inference
- **Training Mode:** DevProxy runs in "Learning Mode" for the first N requests to build a normal cluster baseline.
- **Inference Mode:** Subsequent requests are vectorized and passed to the ONNX model. If the reconstruction error (Autoencoder) or outlier score (Isolation Forest) exceeds a threshold, an Anomaly Finding is generated.

## 🛠️ Implementation Steps
1. **Add ONNX Dependency:** Integrate the Go ONNX runtime.
2. **Feature Vectorizer:** Create `pkg/analysis/ml/vectorizer.go` to convert HTTP requests into `[]float32` tensors.
3. **Model Manager:** Implement a loader that pulls a tiny `<5MB` `.onnx` model file from disk or embedded filesystem.
4. **Integration:** Add the `MLAnomalyRule` to the `SecurityRulesEngine`.

## ⚠️ Security & Constraints
- **Resource Usage:** Inference must be strictly isolated to the asynchronous background workers to ensure the primary proxy data-path remains at zero latency.
- **False Positives:** The ML engine must provide a confidence score. Only high-confidence anomalies should trigger `HIGH` severity alerts in the UI to prevent developer alert fatigue.
