# GoXfer 🔐  
**Zero-Trust, End-to-End Encrypted File Transfer System**

GoXfer is a security-first file transfer system built in Go, designed around **zero-trust principles**, **client-side encryption**, and **minimal attack surface**.  
It consists of a Go backend and a terminal-first (TUI) client, intentionally avoiding bloated UIs and implicit trust in servers.

This is **not** a cloud storage clone. It is a systems-oriented project focused on cryptography, secure client–server design, and real-world deployment constraints.

---

## TL;DR
A zero-trust file transfer system where the server never sees plaintext, authentication avoids offline brute-force oracles, and all critical security decisions happen on the client.

---

## 🎯 Project Goals & Design Principles

- **Zero-Trust Architecture**  
  The server is never trusted with plaintext data or long-term secrets.

- **End-to-End Encryption**  
  Files are encrypted on the client before transmission and decrypted only on the client.

- **Zero-Knowledge Authentication**  
  Authentication is designed to avoid password-derived server-side oracles.

- **Minimal Attack Surface**  
  Lean APIs, limited metadata exposure, and no unnecessary dependencies.

- **Terminal-First UX**  
  A TUI is used instead of a web UI for speed, focus, and scriptability.

- **Production-oriented Design**  
  Built with deployment, failure handling, and extensibility in mind.

---

## 🧠 System Architecture (High Level)

GoXfer is structured as a **single coherent system** split cleanly into components:

- **Client (TUI)**
  - Key management
  - Encryption / decryption
  - Authentication
  - User interaction

- **Backend**
  - Authentication verification
  - Encrypted blob storage
  - Metadata handling
  - Access control & rate limiting

All communication happens over authenticated channels, with the backend acting as a **blind storage and routing layer**.

---

## 🔐 Security Model

### Threats Considered
- Malicious or compromised server
- Database leaks
- Offline brute-force attacks
- Network interception (MITM)
- Replay attempts

### Security Guarantees
- Server never sees plaintext files
- Server never stores password-equivalent secrets
- Encrypted data is useless if exfiltrated
- Authentication resists offline attacks

### Explicit Non-Goals
- Protecting against a compromised client device
- Hardware-level key extraction
- Nation-state adversaries

---

## 🔑 Cryptography Overview

- **Client-Side Encryption**
  - Files are encrypted before leaving the client
- **Key Hierarchy**
  - Data Encryption Keys (DEKs)
  - Key Encryption Keys (KEKs)
  - Multiple wrapping layers
- **No Plaintext Keys at Rest**
- **Key Lifecycle Management**
  - Creation → Use → Zeroization

The design prioritizes **removal of trust**, not just stronger algorithms.

---

## 🔐 Authentication & Authorization

- Authentication follows **zero-knowledge principles**
- Inspired by PAKE / OPAQUE-style approaches
- Prevents offline brute-force even after server compromise
- Session-based access with replay protection
- Clear separation between authentication and encryption keys

---

## 🖥 Backend Responsibilities

- Authentication verification
- Secure API endpoints
- Encrypted file and metadata storage
- Integrity validation
- Abuse prevention and rate limiting
- Logging without leaking sensitive data

The backend treats all stored data as **opaque blobs**.

---

## 💻 TUI Responsibilities

- Secure credential input
- Local key derivation and storage
- Encryption / decryption
- Upload and download orchestration
- Progress tracking and retries
- Atomic file handling on disk

The TUI is designed to be fast, minimal, and distraction-free.

---

## 🔄 Data Flow (Simplified)

1. User authenticates via the TUI
2. Keys are derived locally
3. File is encrypted on the client
4. Encrypted data is transmitted
5. Backend stores opaque ciphertext
6. Download reverses the process

At no point does the server access plaintext.

---

## 🧯 Failure Handling & Reliability

- Network interruption recovery
- Chunked transfers
- Atomic file replacement
- Idempotent operations where possible
- Client-side retries without data corruption

---

## ⚡ Performance Considerations

- Streaming encryption (no full file buffering)
- Controlled memory usage
- Efficient Go concurrency
- Minimal serialization overhead

Security is enforced without turning the system slow or fragile.

---

## 🧱 Project Structure Philosophy

- Modular components
- Clear package boundaries
- No “god modules”
- Easy to extend:
  - New storage backends
  - New authentication methods
  - New client implementations

---

## 🚀 Configuration & Deployment

- Environment-based configuration
- No hard dependency on managed services
- Suitable for:
  - Single VPS
  - Dockerized deployment
  - Cloud VMs
- Secrets never hardcoded

---

## 🚫 What This Project Is NOT

- Not a consumer cloud product
- Not a frontend-heavy app
- Not a crypto library showcase
- Not a CRUD demo
- Not security by obscurity

---

## 📚 Learning Outcomes

- Secure system design
- Applied cryptography in real systems
- Zero-trust architecture
- Go networking and concurrency
- TUI application design
- Failure-aware backend development

---

## 🛣 Future Roadmap

- Distributed backend support
- Stronger PAKE integration
- Audit-friendly logging
- Multi-device key synchronization
- Plugin-based storage layers
- Optional web client (low priority)

---

## 👥 Who This Project Is For

- Backend engineers
- Systems programmers
- Security engineers
- Infrastructure enthusiasts
- Anyone bored of CRUD projects

---

## ⚠ Disclaimer

This project is experimental and educational.  
It has not been professionally audited.  
Use at your own risk.

---
