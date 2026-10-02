#!/usr/bin/env bash
set -euo pipefail
umask 077

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
secrets="$repo_root/secrets"
mkdir -p "$secrets/keys"

if [[ -e "$secrets/keys/public.pem" && ! -e "$secrets/keys/private.pem" ]]; then
  echo 'Cannot restore the private key from an existing public key' >&2
  exit 1
fi

if [[ ! -e "$secrets/keys/private.pem" ]]; then
  openssl genpkey -algorithm Ed25519 -out "$secrets/keys/private.pem"
fi
if [[ ! -e "$secrets/keys/public.pem" ]]; then
  openssl pkey -in "$secrets/keys/private.pem" -pubout -out "$secrets/keys/public.pem"
fi
if [[ ! -e "$secrets/otp.key" ]]; then
  openssl rand -out "$secrets/otp.key" 32
fi
if [[ ! -e "$secrets/outbox.key" ]]; then
  openssl rand -out "$secrets/outbox.key" 32
fi
