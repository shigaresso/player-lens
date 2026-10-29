// Command contractcheck は生成・更新境界の違反を報告する。
package main

import (
	"github.com/shigaresso/player-lens/tools/contractcheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(contractcheck.Analyzer) }
