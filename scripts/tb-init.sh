#!/bin/sh
# HNL Bank - Inicializacion y arranque de TigerBeetle (libro mayor).
# Formatea el cluster en el primer arranque y luego ejecuta "start".
# El data-directory (/data) vive dentro del contenedor (efimero) porque en
# Docker Desktop sobre WSL2 los volumenes montados no soportan el storage de TB.
#
# Nota: se usa seccomp=unconfined en el servicio porque TigerBeetle necesita
# io_uring, que el perfil seccomp por defecto de Docker bloquea.

set -e

DATA_DIR="${DATA_DIR:-/data}"
ADDRESSES="${ADDRESSES:-0.0.0.0:3000}"
CLUSTER_ID="${CLUSTER_ID:-0}"

# Formatear el cluster si no esta inicializado (los errores se ignoran).
/tigerbeetle format --cluster="${CLUSTER_ID}" --replica=0 --replica-count=1 "${DATA_DIR}" 2>/dev/null || true

# Arrancar el nodo reemplazando el proceso (exec).
exec /tigerbeetle start --addresses="${ADDRESSES}" "${DATA_DIR}"
