package protocol

import _ "embed"

//go:embed analysis_guide.md
var analysisGuideInstructions string

const analysisGuideVersion = "1"

//contractcheck:data 埋め込み済みのガイドを JSON へ出力するデータ
type analysisGuide struct {
	Version      string `json:"version"`
	Instructions string `json:"instructions"`
}
