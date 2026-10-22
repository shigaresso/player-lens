package main

import "github.com/shigaresso/player-lens/internal/mcpaccess"

func readMCPAccess(getenv func(string) string) (*mcpaccess.Access, error) {
	return mcpaccess.New(getenv("MCP_PUBLIC_URL"))
}
