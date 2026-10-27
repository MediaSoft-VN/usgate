#!/usr/bin/env python3

import shlex
import sys
from urllib.parse import quote


def parse_value(value: str) -> str:
    try:
        return " ".join(shlex.split(value, comments=False, posix=True))
    except ValueError as error:
        raise SystemExit("NEW_API_ENV contains an invalid environment value") from error


def compose_postgres_dsn(contents: str) -> str:
    postgres_keys = {
        "POSTGRES_HOST",
        "POSTGRES_PORT",
        "POSTGRES_USER",
        "POSTGRES_DB",
        "POSTGRES_PASSWORD",
    }
    values: dict[str, str] = {}
    lines = contents.splitlines()
    for line in lines:
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        if key == "SQL_DSN" or key in postgres_keys:
            values[key] = parse_value(value.strip())

    if values.get("SQL_DSN"):
        return contents

    configured_postgres_keys = postgres_keys.intersection(values)
    if not configured_postgres_keys:
        return contents
    missing_keys = postgres_keys - configured_postgres_keys
    if missing_keys or any(not values.get(key) for key in postgres_keys):
        raise SystemExit("NEW_API_ENV has incomplete PostgreSQL settings")

    host = values["POSTGRES_HOST"]
    port = values["POSTGRES_PORT"]
    if not port.isdecimal() or not 1 <= int(port) <= 65535:
        raise SystemExit("NEW_API_ENV has an invalid POSTGRES_PORT")
    if ":" in host and not host.startswith("["):
        host = f"[{host}]"

    dsn = (
        "postgresql://"
        f"{quote(values['POSTGRES_USER'], safe='')}:"
        f"{quote(values['POSTGRES_PASSWORD'], safe='')}@"
        f"{quote(host, safe='[]:.')}:{port}/"
        f"{quote(values['POSTGRES_DB'], safe='')}"
    )
    output_lines = [
        line
        for line in lines
        if not (
            "=" in line
            and line.split("=", 1)[0].strip() == "SQL_DSN"
            and not parse_value(line.split("=", 1)[1].strip())
        )
    ]
    output = "\n".join(output_lines).rstrip()
    return f"{output}\nSQL_DSN={dsn}\n"


if __name__ == "__main__":
    sys.stdout.write(compose_postgres_dsn(sys.stdin.read()))
