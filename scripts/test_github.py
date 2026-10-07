import unittest
from unittest.mock import Mock

import github
from test_conventions import issue, pr


class CreationTests(unittest.TestCase):
    def setUp(self):
        self.api = Mock(return_value=[{"name": "ci"}])
        self.send = Mock(return_value="https://github.com/example/project/issues/4\n")

    def create(self, kind, data, **kwargs):
        return github.create(kind, "example/project", data,
                             api=self.api, send=self.send, **kwargs)

    def test_valid_issue_sends_the_validated_body(self):
        data = issue()
        self.assertEqual(self.send.return_value, self.create("issue", data))
        self.send.assert_called_once_with(
            ["gh", "issue", "create", "--repo=example/project",
             f"--title={data['title']}", "--body-file=-", "--label=ci"],
            body=data["body"])

    def test_invalid_issue_stops_before_api_or_creation(self):
        with self.assertRaises(ValueError):
            self.create("issue", issue("空欄"))
        self.api.assert_not_called()
        self.send.assert_not_called()

    def test_missing_label_stops_creation(self):
        self.api.return_value = []
        with self.assertRaisesRegex(ValueError, "ラベル"):
            self.create("issue", issue())
        self.send.assert_not_called()


if __name__ == "__main__":
    unittest.main()
