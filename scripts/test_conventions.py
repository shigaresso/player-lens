import unittest

import conventions


class CommitTests(unittest.TestCase):
    def test_valid_message(self):
        """日本語の要約と本文を持つ独自形式が通る。"""
        self.assertEqual([], conventions.validate_commit(
            "(#3) ci: 規約検査を追加した\n\n- 共通の検査を実装した\n"))


    def test_invalid_title(self):
        self.assertTrue(conventions.validate_commit("(#0) unknown: 検査した\n\n- 実装した\n"))


    def test_missing_blank_line(self):
        self.assertTrue(conventions.validate_commit("(#3) ci: 検査した\n- 実装した\n"))


    def test_missing_bullet(self):
        self.assertTrue(conventions.validate_commit("(#3) ci: 検査した\n\n対応した\n"))


    def test_literal_newline(self):
        self.assertTrue(conventions.validate_commit("(#3) ci: 検査した\n\n- 改行\\nを記載した\n"))


    def test_english_only_title(self):
        self.assertTrue(conventions.validate_commit("(#3) ci: Added validation\n\n- 検査を追加した\n"))


    def test_example_bullet_does_not_count(self):
        self.assertTrue(conventions.validate_commit(
            "(#3) ci: 検査した\n\n```text\n- 実装した\n```\n"))


PR_BODY = """## 背景・目的
記載漏れを防ぐ。

## 変更内容
- 検査を追加した。

## 検証
関連テストが成功した。

## 参照仕様・ADR
AGENTS.md のコミット規約を確認した。変更対象にアプリケーション仕様は該当しない。

## 関連 Issue
Closes #3
"""


def pr(body=PR_BODY, title="規約検査を追加した", labels=None):
    return {"title": title, "body": body, "labels": ["ci"] if labels is None else labels}


class PullRequestTests(unittest.TestCase):
    def test_valid_pr(self):
        self.assertEqual([], conventions.validate_pr(pr()))


    def test_empty_section(self):
        self.assertTrue(conventions.validate_pr(pr(PR_BODY.replace("- 検査を追加した。", "<!-- 記載する -->\n- [ ]\nTODO"))))


    def test_duplicate_section(self):
        self.assertTrue(conventions.validate_pr(pr(PR_BODY + "\n## 検証\n成功した。\n")))


    def test_prefixed_title(self):
        self.assertTrue(conventions.validate_pr(pr(title="feat: 規約検査を追加した")))


    def test_non_english_label(self):
        self.assertTrue(conventions.validate_pr(pr(labels=["検査"])))


    def test_example_issue_reference(self):
        self.assertTrue(conventions.validate_pr(pr(PR_BODY.replace("Closes #3", "```text\nCloses #3\n```\n<!-- Closes #3 -->\n説明だけ"))))


ISSUE_BODY = """## 背景・目的
規約を検査する。
## 対応内容
共通検査を実装する。
## 入力・正常な結果・想定する失敗
正常な本文は通り、空欄は失敗する。純粋な検査に I/O を渡さない。
## 完了条件
- [ ] 正常例と違反例を検査できる。
## 検証方法
unittest で正常例と違反例を確認する。
## 参照仕様・ADR
AGENTS.md の自動検査を参照する。
## 依存する Issue
なし。
## 今回扱わない範囲
アプリケーションの実装。
"""


def issue(body=ISSUE_BODY):
    return {"title": "規約検査を導入する", "body": body, "labels": [{"name": "ci"}]}


class IssueTests(unittest.TestCase):
    def test_valid_issue(self):
        self.assertEqual([], conventions.validate_issue(issue()))


    def test_completed_conditions(self):
        self.assertEqual([], conventions.validate_issue(issue(ISSUE_BODY.replace("[ ]", "[x]"))))

    def test_conditions_require_checklist(self):
        self.assertTrue(conventions.validate_issue(issue(ISSUE_BODY.replace("- [ ]", "文章だけ"))))


class ContentTests(unittest.TestCase):
    def test_empty_markers(self):
        self.assertFalse(conventions.has_content("-\n*\n---\n[ ]\n**TODO**\n> TODO"))


if __name__ == "__main__":
    unittest.main()
