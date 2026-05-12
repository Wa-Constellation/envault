# envault

Encrypted environment variable manager. Store secrets in profiles, inject them into subprocesses on demand. Secrets never touch disk in plaintext.

## Install

```bash
go install github.com/Wa-Constellation/envault@latest
```

Or build from source:

```bash
git clone https://github.com/Wa-Constellation/envault.git
cd envault
make build
```

## Quick Start

```bash
# Add vars to a profile
envault add myproject DB_HOST=localhost DB_PORT=5432

# Add a secret interactively (won't appear in shell history)
envault add myproject DB_PASSWORD
# Enter value for DB_PASSWORD: ****

# Import from a .env file
envault add myproject --from-file .env

# Capture from current environment
envault add myproject --from-env AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY

# List profiles
envault ls

# List keys in a profile (values are NOT shown)
envault ls myproject

# Run a command with secrets injected
envault exec myproject -- ./deploy.sh
envault exec myproject -- docker compose up

# Export (use exec instead when possible)
envault export myproject                  # shell format
envault export myproject --format dotenv  # .env format
envault export myproject --format json    # JSON format
```

## Profile Inheritance

```bash
envault add base REGION=us-east-1 ENV=base
envault add dev ENV=development DEBUG=true
envault set-inherit dev base

# dev now has: REGION=us-east-1 (inherited), ENV=development (overridden), DEBUG=true
envault exec dev -- env
```

## AWS Role Assumption

If a profile contains `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_ROLE_ARN`, envault automatically calls STS `AssumeRole` before injecting credentials. The subprocess receives short-lived temporary credentials; the long-term keys and role ARN never reach the child.

Without `AWS_ROLE_ARN`, credentials are passed through as-is.

```bash
# Store credentials + role
envault add aws-prod AWS_ACCESS_KEY_ID=AKIA... AWS_SECRET_ACCESS_KEY=... AWS_ROLE_ARN=arn:aws:iam::123456789012:role/MyRole

# Subprocess receives temporary credentials (key + secret + session token)
envault exec aws-prod -- terraform apply

# Skip role assumption
envault exec aws-prod --no-sts -- aws sts get-caller-identity
```

The region for the STS call is picked from `AWS_REGION` or `AWS_DEFAULT_REGION` in the profile, falling back to `us-east-1`.

## TTL (Time-To-Live)

```bash
envault set-ttl dev 8h    # expires 8 hours after last update
envault set-ttl dev 7d    # 7 days
envault set-ttl dev 0     # disable TTL
```

If a profile has expired, `exec` and `export` will refuse to run and prompt you to refresh with `envault add`.

## Backends

**File backend** (default): All profiles stored in a single AES-256-GCM encrypted file (`~/.config/envault/vault.enc`). You'll be prompted for a passphrase on first use.

**Keyring backend**: Uses the OS keyring (macOS Keychain, Linux libsecret/GNOME Keyring).

```bash
# Switch default backend
envault config --set-backend keyring

# Override per-command
envault --backend file add myprofile KEY=VALUE
```

For CI/automation, set `ENVAULT_PASSPHRASE` to skip the interactive prompt (file backend only).

## Removing Profiles and Keys

```bash
envault rm myproject KEY1 KEY2   # remove specific keys
envault rm myproject             # remove entire profile (asks for confirmation)
envault rm myproject --yes       # skip confirmation
```

## Configuration

Config file: `~/.config/envault/config.toml`

```bash
envault config   # print current configuration
```

## Security

- Secrets encrypted at rest with AES-256-GCM
- Key derivation via Argon2id (OWASP recommended parameters)
- Vault file permissions: `0600`, directory: `0700`
- Atomic writes (temp file + rename) to prevent corruption
- Secrets only live in the child process's memory
- `envault add profile KEY=VALUE` appears in shell history — use `--from-file` or interactive prompt for sensitive values

## Build & Test

```bash
make build    # build binary
make test     # run tests
make test-v   # verbose test output
make lint     # run golangci-lint
make clean    # remove binary
```

## License

MIT
