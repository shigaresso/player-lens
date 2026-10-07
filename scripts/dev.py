"""Compose の開発用 HTTP サーバーを空きポートで起動する。"""

import re


def start_port(configuration):
    """start_port は Compose が解決した app のホスト側ポートを返す。"""
    ports = configuration.get("services", {}).get("app", {}).get("ports", [])
    if len(ports) != 1:
        raise ValueError("app の公開ポートは1つ必要となる")
    mapping = ports[0]
    value = str(mapping.get("published", ""))
    if (mapping.get("host_ip") != "127.0.0.1"
            or mapping.get("protocol") != "tcp"
            or not re.fullmatch(r"[0-9]+", value)
            or not 1 <= int(value) <= 65535):
        raise ValueError("DEV_PORT は1〜65535、公開先は127.0.0.1のTCPにする")
    return int(value)
