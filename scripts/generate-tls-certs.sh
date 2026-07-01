#!/usr/bin/env bash

set -euo pipefail

CERT_DIR=".certs"
CERT_DAYS=3650

CA_CERT="${CERT_DIR}/ca.pem"
CA_KEY="${CERT_DIR}/ca-key.pem"

SERVER_CERT="${CERT_DIR}/server.pem"
SERVER_KEY="${CERT_DIR}/server-key.pem"

SERVER_CSR="${CERT_DIR}/server.csr"
SERVER_EXT="${CERT_DIR}/server.ext"

# NOTE: проверяем только наличие файлов, а не их корректность
if [[ -f "${CA_CERT}" &&
	-f "${CA_KEY}" &&
	-f "${SERVER_CERT}" &&
	-f "${SERVER_KEY}" ]]; then
	echo "(^_^) Development TLS certificates already exist"
	exit 0
fi

if ! command -v openssl >/dev/null 2>&1; then
  echo "(+_+) OpenSSL is required to generate Development TLS certificates" >&2
  exit 1
fi

umask 077
mkdir -p "${CERT_DIR}"

rm -f \
	"${CA_CERT}" \
	"${CA_KEY}" \
	"${SERVER_CERT}" \
	"${SERVER_KEY}" \
	"${SERVER_CSR}" \
	"${SERVER_EXT}"

cat >"${SERVER_EXT}" <<'EOF'
[server_cert]
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:localhost,DNS:server,IP:127.0.0.1,IP:::1
EOF

# создать ключ и самоподписанный сертификат центра сертификации
openssl genrsa -out "${CA_KEY}" 2048

openssl req \
	-x509 \
	-new \
	-sha256 \
	-key "${CA_KEY}" \
	-days "${CERT_DAYS}" \
	-subj "/CN=GophKeeper Development CA" \
	-addext "basicConstraints = critical, CA:TRUE" \
	-addext "keyUsage = critical, keyCertSign, cRLSign" \
	-addext "subjectKeyIdentifier = hash" \
	-out "${CA_CERT}"

# создать ключ и запрос на сертификат HTTPS-сервера
openssl genrsa -out "${SERVER_KEY}" 2048

openssl req \
	-new \
	-sha256 \
	-key "${SERVER_KEY}" \
	-subj "/CN=server" \
	-out "${SERVER_CSR}"

# подписать сертификат Сервера созданным центром сертификации
openssl x509 \
	-req \
	-sha256 \
	-in "${SERVER_CSR}" \
	-CA "${CA_CERT}" \
	-CAkey "${CA_KEY}" \
	-set_serial 1 \
	-days "${CERT_DAYS}" \
	-extfile "${SERVER_EXT}" \
	-extensions server_cert \
	-out "${SERVER_CERT}"

rm -f "${SERVER_CSR}" "${SERVER_EXT}"

chmod 600 "${CA_KEY}" "${SERVER_KEY}"
chmod 644 "${CA_CERT}" "${SERVER_CERT}"

openssl verify \
	-x509_strict \
	-CAfile "${CA_CERT}" \
	"${SERVER_CERT}"

echo "(*_*) Development TLS certificates generated in ${CERT_DIR}"
