package protocol

import _ "embed"

//go:embed analysis_guide.md
var analysisGuideInstructions string

const analysisGuideVersion = "1"

type analysisGuide struct {
	Version      string `json:"version"`
	Instructions string `json:"instructions"`
}
