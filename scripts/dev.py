"""Compose の開発用 HTTP サーバーを空きポートで起動する。"""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


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


def require_success(result):
    """require_success は Compose の失敗原因を保持して中断する。"""
    if result.returncode:
        raise RuntimeError((result.stdout or "") + (result.stderr or "")
                           or f"Compose が終了コード{result.returncode}で失敗した")
    return result.stdout


def launch(*, compose, environment, announce=print):
    """launch は作成した app を所有し、Watch 終了時にも停止する。"""
    env = dict(environment)
    configuration = json.loads(require_success(compose("config", "--format", "json", env=env)))
    port = start_port(configuration)
    running = require_success(compose("ps", "--status", "running", "--quiet", "app", env=env))
    if running.strip():
        raise RuntimeError("同じプロジェクトの app は起動済み。既存の起動を停止してから実行する")
    require_success(compose("build", "app", env=env))
    try:
        for port in range(port, 65536):
            env = {**env, "DEV_PORT": str(port)}
            result = compose("up", "--detach", "--wait", "--wait-timeout", "60",
                             "--no-build", "app", env=env)
            if result.returncode == 0:
                break
            error = (result.stdout or "") + (result.stderr or "")
            if ("Error response from daemon:" not in error
                    or not any(reason in error for reason in
                               ("port is already allocated", "bind: address already in use"))):
                require_success(result)
            if port < 65535:
                print(f"ポート{port}は使用中のため、次のポートを試します。", file=sys.stderr, flush=True)
        else:
            raise RuntimeError("65535まで使用中のため、公開できる空きポートがない")
        announce(f"開発サーバー: http://127.0.0.1:{port}（Ctrl+Cで停止）")
        require_success(compose("watch", "--no-up", "app", env=env, capture=False))
        return port
    finally:
        original = sys.exception()
        try:
            require_success(compose("stop", "app", env=env))
        except (OSError, RuntimeError, subprocess.SubprocessError) as error:
            if original is not None:
                raise RuntimeError(f"{original}\napp の停止にも失敗した: {error}") from error
            raise


def run_compose(*args, env, capture=True):
    """run_compose は解決・起動・監視をリポジトリの Compose へ渡す。"""
    timeout = None if args[0] == "watch" else 600 if args[0] == "build" else 90
    return subprocess.run(["docker", "compose", *args], cwd=ROOT, env=env,
                          text=True, capture_output=capture, timeout=timeout, check=False)


def main():
    try:
        launch(compose=run_compose, environment=os.environ,
               announce=lambda message: print(message, flush=True))
    except KeyboardInterrupt:
        return 0
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print(f"開発サーバーの起動・監視に失敗した: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
