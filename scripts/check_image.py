"""本番用イメージを一度ビルドし、同じ image ID の実動作を検証する。"""

import contextlib
import http.client
import json
import subprocess
import sys
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def docker(*args, timeout=30):
    result = subprocess.run(["docker", *args], cwd=ROOT, check=True, text=True,
                            capture_output=True, timeout=timeout)
    return (result.stdout + result.stderr if args[0] == "logs" else result.stdout).strip()


@contextlib.contextmanager
def container(image, *options):
    identifier = docker("run", "--detach", "--read-only", *options, image)
    try:
        yield identifier
    finally:
        docker("rm", "--force", identifier)


def request(port, method, path):
    connection = http.client.HTTPConnection("127.0.0.1", port, timeout=1)
    try:
        connection.request(method, path)
        response = connection.getresponse()
        return response.status, response.getheader("Content-Type"), response.read()
    finally:
        connection.close()


def check_running(image, port, signal, *environment):
    with container(image, "--publish", f"127.0.0.1::{port}", *environment) as identifier:
        state = json.loads(docker("inspect", identifier))[0]
        if state["Image"] != image or state["Mounts"]:
            raise RuntimeError("検証対象が指定したイメージと一致しないか、共有が存在する")
        host_port = int(docker("port", identifier, f"{port}/tcp").rsplit(":", 1)[1])
        deadline = time.monotonic() + 10
        while True:
            try:
                response = request(host_port, "GET", "/healthz")
                break
            except (OSError, http.client.HTTPException):
                if time.monotonic() >= deadline:
                    raise RuntimeError("healthz が起動期限内に応答しなかった")
                if docker("inspect", "--format", "{{.State.Running}}", identifier) != "true":
                    raise RuntimeError("サーバーが起動しなかった: " + docker("logs", identifier))
                time.sleep(0.1)
        if response != (200, "text/plain; charset=utf-8", b"ok\n"):
            raise RuntimeError(f"healthz の応答が異なる: {response!r}")
        if request(host_port, "HEAD", "/healthz") != (200, "text/plain; charset=utf-8", b""):
            raise RuntimeError("HEAD の応答が異なる")
        for method, path, status in (("POST", "/healthz", 405), ("GET", "/", 404)):
            if request(host_port, method, path)[0] != status:
                raise RuntimeError(f"{method} {path} の状態が異なる")
        docker("kill", "--signal", signal, identifier)
        if docker("wait", identifier, timeout=15) != "0":
            raise RuntimeError("終了シグナルで正常終了しなかった")
        print(f"PORT={port}: HTTP 応答と {signal} の正常終了を確認した")


def main():
    try:
        with tempfile.TemporaryDirectory(prefix="player-lens-image-") as directory:
            iid = Path(directory) / "image-id"
            docker("build", "--target", "production", "--iidfile", str(iid), ".", timeout=600)
            image = iid.read_text(encoding="utf-8").strip()
            config = json.loads(docker("image", "inspect", "--format", "{{json .Config}}", image))
            if config["User"] != "65532:65532":
                raise RuntimeError("本番用イメージが非 root の設定ではない")
            print(f"検証対象の image ID: {image}", flush=True)
            check_running(image, 8080, "SIGTERM")
            check_running(image, 10000, "SIGINT", "--env", "PORT=10000")
            with container(image, "--env", "PORT=invalid") as identifier:
                if docker("wait", identifier, timeout=15) == "0":
                    raise RuntimeError("不正な PORT で正常終了した")
                if "PORT" not in docker("logs", identifier):
                    raise RuntimeError("不正な PORT の原因を報告しなかった")
            print("不正な PORT の非0終了を確認した")
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print(f"本番用イメージ検証が失敗した: {error}", file=sys.stderr)
        if isinstance(error, subprocess.CalledProcessError):
            print(error.stderr[-3000:], file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
