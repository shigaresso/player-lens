"""使い捨て Git リポジトリの検査を呼び出し元の Git 環境から隔離する。"""

from contextlib import contextmanager
import os
from unittest.mock import patch


@contextmanager
def isolated_git_environment():
    """一時リポジトリだけを操作し、正常終了・例外の両方で元の環境を復元する。"""
    environment = {key: value for key, value in os.environ.items()
                   if not key.startswith("GIT_")}
    environment.update(GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull)
    with patch.dict(os.environ, environment, clear=True):
        yield
