#!/usr/bin/env python3

import grp
import os
import re
import socket
import socketserver
import stat
import struct
import subprocess

SOCKET_PATH = "/run/new-api-tenant-domain.sock"
HELPER_PATH = "/usr/local/sbin/new-api-tenant-domain"
MAX_REQUEST_BYTES = 512
DOMAIN_LABEL = re.compile(r"^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")


def valid_hostname(hostname):
    if not hostname or len(hostname) > 253:
        return False
    return all(DOMAIN_LABEL.fullmatch(label) for label in hostname.split("."))


class DomainProvisionerHandler(socketserver.StreamRequestHandler):
    def handle(self):
        expected_uid = int(os.environ["NEW_API_DOMAIN_APP_UID"])
        peer_pid, peer_uid, _ = struct.unpack(
            "3i", self.request.getsockopt(socket.SOL_SOCKET, socket.SO_PEERCRED, struct.calcsize("3i"))
        )
        del peer_pid
        line = self.rfile.readline(MAX_REQUEST_BYTES + 1)
        if peer_uid != expected_uid or len(line) > MAX_REQUEST_BYTES or not line.endswith(b"\n"):
            self.wfile.write(b"ERROR\n")
            return
        try:
            operation, hostname = line.decode("ascii").rstrip("\n").split("\t")
        except (UnicodeDecodeError, ValueError):
            self.wfile.write(b"ERROR\n")
            return
        if operation not in ("check", "provision", "cleanup") or not valid_hostname(hostname):
            self.wfile.write(b"ERROR\n")
            return
        try:
            result = subprocess.run(
                [HELPER_PATH, operation, hostname],
                stdin=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
                timeout=540,
            )
        except (OSError, subprocess.TimeoutExpired):
            self.wfile.write(b"ERROR\n")
            return
        self.wfile.write(b"OK\n" if result.returncode == 0 else b"ERROR\n")


class DomainProvisionerServer(socketserver.UnixStreamServer):
    request_queue_size = 4


def main():
    if os.geteuid() != 0:
        raise SystemExit(1)
    app_group = grp.getgrnam(os.environ["NEW_API_DOMAIN_APP_GROUP"])
    try:
        existing = os.lstat(SOCKET_PATH)
    except FileNotFoundError:
        pass
    else:
        if not stat.S_ISSOCK(existing.st_mode) or existing.st_uid != 0:
            raise SystemExit(1)
        os.unlink(SOCKET_PATH)

    server = DomainProvisionerServer(SOCKET_PATH, DomainProvisionerHandler)
    os.chown(SOCKET_PATH, 0, app_group.gr_gid)
    os.chmod(SOCKET_PATH, 0o660)
    try:
        server.serve_forever()
    finally:
        server.server_close()
        if os.path.exists(SOCKET_PATH) and stat.S_ISSOCK(os.lstat(SOCKET_PATH).st_mode):
            os.unlink(SOCKET_PATH)


if __name__ == "__main__":
    main()
