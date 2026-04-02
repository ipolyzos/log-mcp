package security

import (
	"os"
	"path/filepath"
	"strings"
)

// FileReadableUnderRoots reports whether targetAbs is a regular file whose
// canonical path (after symlink resolution) stays under one of the roots.
// Each root should be an absolute clean path (as from [filepath.Clean] on [filepath.Abs]).
func FileReadableUnderRoots(roots []string, targetAbs string) bool {
	if len(roots) == 0 {
		return false
	}
	targetAbs = filepath.Clean(targetAbs)
	realTarget, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return false
	}
	fi, err := os.Stat(realTarget)
	if err != nil || fi.IsDir() || !fi.Mode().IsRegular() {
		return false
	}
	realTarget = filepath.Clean(realTarget)
	for _, root := range roots {
		root = filepath.Clean(root)
		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		realRoot = filepath.Clean(realRoot)
		if underRoot(realRoot, realTarget) {
			return true
		}
	}
	return false
}

func underRoot(rootAbs, pathAbs string) bool {
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}
