import json
import subprocess
import unittest
from unittest.mock import Mock, patch

import dev


class ConfigurationTests(unittest.TestCase):
    def test_reads_resolved_host_port(self):
        configuration = {"services": {"app": {"ports": [{
            "host_ip": "127.0.0.1", "published": "18080",
            "target": 8080, "protocol": "tcp",
        }]}}}
        self.assertEqual(18080, dev.start_port(configuration))

    def test_rejects_invalid_ports_or_unsupported_mappings(self):
        cases = {
            "0は拒否する": [{"host_ip": "127.0.0.1", "published": "0", "protocol": "tcp"}],
            "上限超過は拒否する": [{"host_ip": "127.0.0.1", "published": "65536", "protocol": "tcp"}],
            "非数字は拒否する": [{"host_ip": "127.0.0.1", "published": "invalid", "protocol": "tcp"}],
            "動的割り当ては拒否する": [{"host_ip": "127.0.0.1", "protocol": "tcp"}],
            "全インターフェース公開は拒否する": [{"host_ip": "0.0.0.0", "published": "8080", "protocol": "tcp"}],
            "UDPは拒否する": [{"host_ip": "127.0.0.1", "published": "8080", "protocol": "udp"}],
            "複数ポートは拒否する": [{}, {}],
            "公開なしは拒否する": [],
        }
        for name, ports in cases.items():
            with self.subTest(name):
                with self.assertRaises(ValueError):
                    dev.start_port({"services": {"app": {"ports": ports}}})


class LaunchTests(unittest.TestCase):
    def setUp(self):
        config = {"services": {"app": {"ports": [{
            "host_ip": "127.0.0.1", "published": "8080",
            "target": 10000, "protocol": "tcp",
        }]}}}
        self.environment = {"PORT": "10000", "DEV_PORT": "8080"}
        self.compose = Mock(side_effect=[self.result(json.dumps(config)),
                                        self.result(), self.result(), self.result(),
                                        self.result(), self.result()])
        self.announce = Mock()

    @staticmethod
    def result(stdout="", stderr="", returncode=0):
        return subprocess.CompletedProcess([], returncode, stdout, stderr)

    def launch(self):
        with patch("sys.stderr"):
            return dev.launch(compose=self.compose, environment=self.environment,
                              announce=self.announce)

    def test_starts_watches_and_stops_without_changing_container_port(self):
        self.assertEqual(8080, self.launch())
        self.announce.assert_called_once_with("開発サーバー: http://127.0.0.1:8080（Ctrl+Cで停止）")
        commands = [call.args for call in self.compose.call_args_list]
        self.assertEqual([
            ("config", "--format", "json"),
            ("ps", "--status", "running", "--quiet", "app"),
            ("build", "app"),
            ("up", "--detach", "--wait", "--wait-timeout", "60", "--no-build", "app"),
            ("watch", "--no-up", "app"),
            ("stop", "app"),
        ], commands)
        self.assertEqual({"PORT": "10000", "DEV_PORT": "8080"}, self.environment)
        for call in self.compose.call_args_list:
            self.assertEqual("10000", call.kwargs["env"]["PORT"])

    def test_retries_only_occupied_host_ports_in_order(self):
        initial = list(self.compose.side_effect)
        self.compose.side_effect = initial[:3] + [
            self.result(stderr="Error response from daemon: port is already allocated", returncode=1),
            self.result(stderr="Error response from daemon: bind: address already in use", returncode=1),
        ] + initial[3:]
        self.assertEqual(8082, self.launch())
        attempts = [call.kwargs["env"]["DEV_PORT"] for call in self.compose.call_args_list
                    if "--detach" in call.args]
        self.assertEqual(["8080", "8081", "8082"], attempts)
        self.announce.assert_called_once_with("開発サーバー: http://127.0.0.1:8082（Ctrl+Cで停止）")

    def test_other_start_failures_do_not_retry_and_stop_owned_app(self):
        for error in ("container app is unhealthy", "Cannot connect to the Docker daemon",
                      "application log: bind: address already in use"):
            with self.subTest(error):
                self.setUp()
                results = list(self.compose.side_effect)
                self.compose.side_effect = results[:3] + [self.result(stderr=error, returncode=1),
                                                         self.result()]
                with self.assertRaisesRegex(RuntimeError, error):
                    self.launch()
                self.assertEqual(1, sum("--detach" in call.args
                                        for call in self.compose.call_args_list))
                self.assertEqual(("stop", "app"), self.compose.call_args.args)
                self.announce.assert_not_called()

    def test_port_exhaustion_does_not_wrap_or_start_watch(self):
        results = list(self.compose.side_effect)
        config = json.loads(results[0].stdout)
        config["services"]["app"]["ports"][0]["published"] = "65535"
        self.compose.side_effect = [self.result(json.dumps(config))] + results[1:3] + [
            self.result(stderr="Error response from daemon: port is already allocated", returncode=1),
            self.result()]
        with self.assertRaisesRegex(RuntimeError, "空きポートがない"):
            self.launch()
        attempts = [call.kwargs["env"]["DEV_PORT"] for call in self.compose.call_args_list
                    if "--detach" in call.args]
        self.assertEqual(["65535"], attempts)

    def test_existing_app_is_not_rebuilt_moved_or_stopped(self):
        results = list(self.compose.side_effect)
        self.compose.side_effect = [results[0], self.result("existing-container\n")]
        with self.assertRaisesRegex(RuntimeError, "起動済み"):
            self.launch()
        self.assertEqual(2, self.compose.call_count)

    def test_build_failure_does_not_attempt_start_or_stop(self):
        results = list(self.compose.side_effect)
        self.compose.side_effect = results[:2] + [self.result(stderr="build failed", returncode=1)]
        with self.assertRaisesRegex(RuntimeError, "build failed"):
            self.launch()
        self.assertEqual(3, self.compose.call_count)

    def test_watch_interrupt_or_failure_stops_owned_app(self):
        for outcome in (KeyboardInterrupt(), self.result(stderr="watch failed", returncode=1)):
            with self.subTest(type(outcome).__name__):
                self.setUp()
                results = list(self.compose.side_effect)
                self.compose.side_effect = results[:4] + [outcome, self.result()]
                with self.assertRaises(KeyboardInterrupt if isinstance(outcome, KeyboardInterrupt)
                                       else RuntimeError):
                    self.launch()
                self.assertEqual(("stop", "app"), self.compose.call_args.args)

    def test_configuration_failure_stops_before_build(self):
        self.compose.side_effect = [self.result(stderr="invalid Compose configuration", returncode=1)]
        with self.assertRaisesRegex(RuntimeError, "invalid Compose configuration"):
            self.launch()
        self.assertEqual(1, self.compose.call_count)

    def test_cli_returns_nonzero_for_launch_failure(self):
        with patch.object(dev, "launch", side_effect=RuntimeError("起動失敗")), \
                patch("sys.stderr") as stderr:
            self.assertEqual(1, dev.main())
            self.assertIn("起動失敗", str(stderr.write.call_args_list))

    def test_stop_failure_preserves_original_watch_error(self):
        results = list(self.compose.side_effect)
        self.compose.side_effect = results[:4] + [
            self.result(stderr="watch failed", returncode=1),
            self.result(stderr="stop failed", returncode=1)]
        with self.assertRaises(RuntimeError) as raised:
            self.launch()
        self.assertIn("watch failed", str(raised.exception))
        self.assertIn("stop failed", str(raised.exception))


if __name__ == "__main__":
    unittest.main()
