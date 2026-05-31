package duplicate

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func NormalizeContent(content string) string {
	trimmed := strings.TrimSpace(content)
	normalizedLines := strings.Split(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\n")
	for index := range normalizedLines {
		normalizedLines[index] = strings.TrimRight(normalizedLines[index], " \t")
	}
	return strings.Join(normalizedLines, "\n")
}

func HashContent(content string) string {
	sum := sha256.Sum256([]byte(NormalizeContent(content)))
	return hex.EncodeToString(sum[:])
}
