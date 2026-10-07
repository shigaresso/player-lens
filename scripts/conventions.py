"""通信や Git に依存しない、開発規約の形式検査。"""

import re

TYPES = "feat|fix|docs|test|refactor|perf|style|build|ci|chore|revert"
COMMIT_TITLE = re.compile(rf"^\(#([1-9][0-9]*)\) ({TYPES}): \S.*$")
JAPANESE = re.compile(r"[ぁ-んァ-ヶ一-龯]")


def visible_lines(text):
    """HTML コメントと fenced code の例示を本文から除く。"""
    fence = None
    for line in re.sub(r"<!--[\s\S]*?(?:-->|$)", "", text).splitlines():
        marker = re.match(r"^\s{0,3}(`{3,}|~{3,})(.*)$", line)
        if marker:
            if fence is None:
                fence = marker[1]
            elif marker[1][0] == fence[0] and len(marker[1]) >= len(fence) and not marker[2].strip():
                fence = None
            continue
        if fence is None:
            yield line


def validate_commit(message):
    """validate_commit はコミット本文の違反を返す。"""
    lines = message.replace("\r\n", "\n").split("\n")
    errors = []
    if not COMMIT_TITLE.fullmatch(lines[0]):
        errors.append("タイトルを (#Issue番号) 種別: 要約 の形式にする")
    if len(lines) < 2 or lines[1] != "":
        errors.append("タイトルと本文の間に空行を入れる")
    if not any(re.fullmatch(r"- +\S.*", line) for line in visible_lines("\n".join(lines[2:]))):
        errors.append("本文に内容のある箇条書きを記載する")
    if "\\n" in message or "\r" in message.replace("\r\n", "\n"):
        errors.append("改行には LF または CRLF を使い、文字列の \\n を含めない")
    if not JAPANESE.search(lines[0]):
        errors.append("要約に日本語を記載する")
    return errors


def validate_pr(data):
    """validate_pr は PR の記載漏れと形式違反を返す。"""
    _, errors = validate_sections(data.get("body") or "", PR_SECTIONS)
    errors.extend(validate_title(data.get("title") or ""))
    errors.extend(validate_labels(data.get("labels") or []))
    if not related_issues(data.get("body") or ""):
        errors.append("関連 Issue の節に Closes #番号 または Refs #番号 を独立した行で記載する")
    if "\\n" in (data.get("body") or ""):
        errors.append("本文に文字列の \\n を含めない")
    if re.match(rf"^(?:\(#[0-9]+\)|(?:{TYPES})(?:\([^)]*\))?!?:)", (data.get("title") or "").strip()):
        errors.append("PR タイトルに Issue 番号・コミット種別の接頭辞を付けない")
    return errors


def validate_title(title):
    if not title.strip() or "\n" in title or "\r" in title or "\\n" in title:
        return ["タイトルに内容を記載し、改行を含めない"]
    if not JAPANESE.search(title):
        return ["タイトルに日本語を記載する"]
    return []


def validate_labels(labels):
    names = [item.get("name", "") if isinstance(item, dict) else item for item in labels]
    if not names or any(not isinstance(name, str) or not re.fullmatch(r"[\x20-\x7e]+", name) or not re.search(r"[A-Za-z]", name) for name in names):
        return ["英字を含む ASCII のラベルを1件以上付ける"]
    return []


def related_issues(body):
    sections, _ = validate_sections(body, ())
    return sorted({int(match[1]) for line in sections.get("関連 Issue", "").splitlines()
                   if (match := re.fullmatch(r"(?:Closes|Refs) #([1-9][0-9]*)", line.strip()))})


ISSUE_SECTIONS = ("背景・目的", "対応内容", "入力・正常な結果・想定する失敗", "完了条件",
                  "検証方法", "参照仕様・ADR", "依存する Issue", "今回扱わない範囲")


def validate_issue(data):
    """validate_issue は進捗の状態に依存しない記載規約を確認する。"""
    sections, errors = validate_sections(data.get("body") or "", ISSUE_SECTIONS)
    errors.extend(validate_title(data.get("title") or ""))
    errors.extend(validate_labels(data.get("labels") or []))
    if not any(re.fullmatch(r"\s*- \[[ xX]\] +\S.*", line) and has_content(line)
               for line in sections.get("完了条件", "").splitlines()):
        errors.append("完了条件に内容を持つチェックリストを記載する（チェック済みも可）")
    if "\\n" in (data.get("body") or ""):
        errors.append("本文に文字列の \\n を含めない")
    return errors


PR_SECTIONS = ("背景・目的", "変更内容", "検証", "参照仕様・ADR", "関連 Issue")
PLACEHOLDERS = {"todo", "tbd", "未記入", "記載する", "_no response_", "no response"}


def has_content(text):
    for line in visible_lines(text):
        content = re.sub(r"^\s*(?:[-*+]\s+|[0-9]+[.)]\s+)?(?:\[[ xX]\]\s*)?", "", line).strip()
        normalized = content.strip("*_`>.。 ").casefold()
        if normalized and not re.fullmatch(r"[-*_+]+", normalized) and not content.startswith("#") and normalized not in PLACEHOLDERS:
            return True
    return False


def validate_sections(body, required):
    sections = {}
    current = None
    errors = []
    for line in visible_lines(body):
        heading = re.fullmatch(r"## +(.+?)\s*", line)
        if heading:
            current = heading[1]
            if current in sections:
                errors.append(f"「{current}」の見出しを重複させない")
            sections.setdefault(current, [])
        elif current:
            sections[current].append(line)
    sections = {name: "\n".join(lines) for name, lines in sections.items()}
    for name in required:
        if name not in sections or not has_content(sections[name]):
            errors.append(f"「{name}」に実質的な内容を記載する")
    return sections, errors
