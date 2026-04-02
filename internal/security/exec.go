package security

import "os/exec"

// HardenExecCmd sets subprocess options for read-only, non-interactive execution:
// stdin is detached so the child cannot read from the MCP server's tty.
func HardenExecCmd(cmd *exec.Cmd) {
	cmd.Stdin = nil
}
