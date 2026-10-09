# Kumokura Security Architecture

## 1. Zero-Plaintext Credential Storage

Kumokura strictly guarantees that S3 access keys, secret keys, and session tokens are **never stored in plaintext on disk**:

1. **Primary Vault (OS Keyring)**:
   - On macOS: Apple Keychain.
   - On Windows: Windows Credential Manager (`wincred`).
   - On Linux Desktop: Secret Service API via D-Bus / GNOME Keyring / KWallet.

2. **Encrypted Fallback (Headless / Containerized Environments)**:
   - When running on headless Linux servers or CI/CD pipelines lacking D-Bus, Kumokura automatically falls back to an encrypted local file vault (`~/.config/kumokura/secrets/vault.enc`).
   - The file is encrypted using **AES-256-GCM** with authenticated data and unique nonces generated from `crypto/rand`.

## 2. SQLite Ledger Encryption & Security

- SQLite database files are initialized with POSIX mode `0600` (readable and writable only by the executing user).
- Application directories (`~/.config/kumokura/`) are initialized with POSIX mode `0700`.

## 3. Network & Transport Security

- TLS 1.2 is enforced as the minimum allowable cryptographic protocol version.
- System CA pools are verified by default.
- Custom enterprise root certificates can be supplied via configuration without bypassing TLS verification.
