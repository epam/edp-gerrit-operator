#!/usr/bin/env bash
#
# Generate the full SSH host key set for the gerrit.sshHostKeys chart feature.
#
# Produces every key type Gerrit would otherwise generate at init (any type
# left out would be created randomly and stay unpinned), plus ready-to-apply
# artifacts built from them:
#
#   <out>/ssh_host_*_key[.pub]  the key pairs (Secret content)
#   <out>/secret.yaml           Kubernetes Secret manifest for the chart
#   <out>/known_hosts           pinned entries for SSH clients (ArgoCD,
#                               codebase-operator knownHosts.entries), when
#                               --host is given
#
# The private keys let their holder impersonate this Gerrit: use them in test
# environments only and keep them out of git. The known_hosts file contains
# only public halves and is safe to commit.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: generate-ssh-host-keys.sh [options]

Options:
  -o, --output DIR      Output directory (default: ./gerrit-ssh-host-keys).
                        Keys are generated only if the directory holds none;
                        an existing set is reused, so one key set can serve
                        any number of namespaces/hosts.
  -s, --secret NAME     Secret name in the manifest (default: gerrit-ssh-host-keys,
                        the chart's gerrit.sshHostKeys.secret default)
  -n, --namespace NS    Namespace in the manifest (default: omitted, so the
                        manifest applies to the current kubectl namespace)
  -H, --host HOST       Gerrit SSH host, e.g. gerrit.my-namespace; enables
                        known_hosts generation
  -p, --port PORT       Gerrit SSH port for known_hosts entries (default: 22;
                        any other port uses the [host]:port bracket form)
  -h, --help            Show this help

Examples:
  # generate the single key set once (e.g. to load into an external secret store)
  generate-ssh-host-keys.sh -o ~/gerrit-keys
  # derive pinned known_hosts entries per namespace from the same set;
  # entries accumulate in one known_hosts file, stale ones are replaced
  generate-ssh-host-keys.sh -o ~/gerrit-keys -H gerrit.ns-a -p 30022
  generate-ssh-host-keys.sh -o ~/gerrit-keys -H gerrit.ns-b -p 30022
EOF
}

out_dir="./gerrit-ssh-host-keys"
secret_name="gerrit-ssh-host-keys"
namespace=""
host=""
port="22"

while [ $# -gt 0 ]; do
  case "$1" in
    -o|--output)    out_dir="$2"; shift 2 ;;
    -s|--secret)    secret_name="$2"; shift 2 ;;
    -n|--namespace) namespace="$2"; shift 2 ;;
    -H|--host)      host="$2"; shift 2 ;;
    -p|--port)      port="$2"; shift 2 ;;
    -h|--help)      usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
  esac
done

for tool in ssh-keygen kubectl; do
  command -v "$tool" >/dev/null || { echo "Error: $tool not found in PATH" >&2; exit 1; }
done

# An existing key set is never overwritten: the whole point of pinned keys is
# that they stay stable. Rerunning against an existing directory only derives
# artifacts (known_hosts for another host, the Secret manifest) from it.
if ls "$out_dir"/ssh_host_* >/dev/null 2>&1; then
  echo "Reusing existing host keys in $out_dir (only deriving artifacts)."
else
  mkdir -p "$out_dir"
  # The set Gerrit (Apache MINA sshd) generates on `gerrit init`.
  echo "Generating host keys in $out_dir ..."
  ssh-keygen -q -t ed25519        -N "" -C gerrit -f "$out_dir/ssh_host_ed25519_key"
  ssh-keygen -q -t rsa   -b 4096  -N "" -C gerrit -f "$out_dir/ssh_host_rsa_key"
  ssh-keygen -q -t ecdsa -b 256   -N "" -C gerrit -f "$out_dir/ssh_host_ecdsa_key"
  ssh-keygen -q -t ecdsa -b 384   -N "" -C gerrit -f "$out_dir/ssh_host_ecdsa_384_key"
  ssh-keygen -q -t ecdsa -b 521   -N "" -C gerrit -f "$out_dir/ssh_host_ecdsa_521_key"
fi

from_files=()
for f in "$out_dir"/ssh_host_*key "$out_dir"/ssh_host_*key.pub; do
  from_files+=("--from-file=$f")
done
ns_arg=()
[ -n "$namespace" ] && ns_arg=(--namespace "$namespace")
# ${arr[@]+...} keeps empty arrays from tripping `set -u` on bash 3.2 (macOS).
kubectl create secret generic "$secret_name" ${ns_arg[@]+"${ns_arg[@]}"} "${from_files[@]}" \
  --dry-run=client -o yaml > "$out_dir/secret.yaml"

if [ -n "$host" ]; then
  if [ "$port" = "22" ]; then
    host_entry="$host"
  else
    host_entry="[$host]:$port"
  fi
  # Upsert: keep entries for other hosts so one file accumulates every
  # namespace, replace any stale entries for this one.
  touch "$out_dir/known_hosts"
  grep -vF "$host_entry " "$out_dir/known_hosts" > "$out_dir/known_hosts.tmp" || true
  for pub in "$out_dir"/ssh_host_*_key.pub; do
    # Drop the trailing comment so entries are exactly "host type key".
    read -r key_type key_data _ < "$pub"
    echo "$host_entry $key_type $key_data" >> "$out_dir/known_hosts.tmp"
  done
  mv "$out_dir/known_hosts.tmp" "$out_dir/known_hosts"
fi

echo
echo "Fingerprints:"
for pub in "$out_dir"/ssh_host_*_key.pub; do
  ssh-keygen -lf "$pub"
done
echo
echo "Wrote:"
echo "  $out_dir/secret.yaml (apply, then set gerrit.sshHostKeys.enabled=true)"
if [ -n "$host" ]; then
  echo "  $out_dir/known_hosts (entries for ArgoCD / codebase-operator knownHosts.entries)"
else
  echo "  (no known_hosts written; pass --host/--port to generate pinned client entries)"
fi
