# Security Policy

## Reporting a Vulnerability

If you discover a security issue in envault, please report it privately to
**contact@wa-systems.eu**. Do not open a public GitHub issue.

Include:

- A description of the issue and its impact
- Steps to reproduce, or a proof-of-concept
- The affected version (output of `envault --version`) and your OS

We will acknowledge your report within 5 business days, work with you to
verify and reproduce the issue, and coordinate a fix and disclosure timeline.

## Supported Versions

Only the most recent minor release receives security fixes. Once `v1.0.0` is
out, we will maintain the latest two minor releases.

## Scope

In scope:

- The envault CLI and its libraries (`internal/...`)
- File and keyring backends, including the on-disk vault format
- Key derivation, encryption, and decryption code paths
- Subprocess environment construction in `envault exec`

Out of scope:

- Vulnerabilities in third-party dependencies (please report those upstream,
  but feel free to CC us if envault's usage exposes them)
- Issues that require an attacker to already have read access to the user's
  decrypted vault, shell history, or process memory
- Issues stemming from misconfiguration of the OS keyring or filesystem
  permissions outside envault's control

## Threat Model

envault assumes:

- The user's passphrase (or `ENVAULT_PASSPHRASE`) is not known to the attacker
- The user's machine is not compromised at the time of vault operations
- The OS keyring, when used, provides confidentiality and integrity equivalent
  to its documented guarantees on the host platform

envault does **not** defend against:

- A compromised host with kernel- or root-level access
- Attacks that observe process memory of envault or its child processes
- Side-channel attacks against the underlying crypto primitives
