"""Issue・PR の作成手順。検査と GitHub への書き込みを同じ入口で扱う。"""

import conventions


def create(kind, repo, data, *, api, send):
    """create は記載内容を受け取り、作成した URL の出力を返す。"""
    errors = conventions.validate_issue(data)
    if errors:
        raise ValueError("; ".join(errors))
    labels = [label["name"] if isinstance(label, dict) else label
              for label in data["labels"]]
    available = {label["name"] for label in api(f"repos/{repo}/labels", paginate=True)}
    if set(labels) - available:
        raise ValueError("指定したラベルが存在しない。作成前にラベルを整備する")
    args = ["gh", kind, "create", f"--repo={repo}",
            f"--title={data['title']}", "--body-file=-"]
    args.extend(f"--label={label}" for label in labels)
    return send(args, body=data["body"])
