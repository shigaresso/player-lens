import unittest

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


if __name__ == "__main__":
    unittest.main()
