package contractcheck_test

import (
	"testing"

	"github.com/shigaresso/player-lens/tools/contractcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestConstructionBoundary(t *testing.T) {
	t.Run("定義ファイル以外の直接生成を拒否し生成済みのコピーを許可する", func(t *testing.T) {
		analysistest.Run(t, analysistest.TestData(), contractcheck.Analyzer, "direct", "consumer")
	})
}

func TestImplicitConstruction(t *testing.T) {
	t.Run("省略フィールドや配列などからの未初期化値の生成を拒否する", func(t *testing.T) {
		analysistest.Run(t, analysistest.TestData(), contractcheck.Analyzer, "implicit")
	})
}

func TestUpdateBoundary(t *testing.T) {
	t.Run("定義側の更新を許可して外側からの更新と内部参照の流出を拒否する", func(t *testing.T) {
		analysistest.Run(t, analysistest.TestData(), contractcheck.Analyzer, "update")
	})
}

func TestControlledDeclaration(t *testing.T) {
	t.Run("公開フィールドを持つ型には生成更新の保証を付けられない", func(t *testing.T) {
		analysistest.Run(t, analysistest.TestData(), contractcheck.Analyzer, "declaration")
	})
}
