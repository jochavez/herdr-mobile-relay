package coordinator

import "testing"

func TestPromptViaPaneTextCoversQoderAndPrime(t *testing.T) {
	for _, agent := range []string{"qoder", "qodercli", "prime-agent", "Prime-Agent", "primeagent"} {
		if !promptViaPaneText(agent) {
			t.Errorf("promptViaPaneText(%q) = false, want true", agent)
		}
	}
	for _, agent := range []string{"claude", "codex", "pi", "omp", ""} {
		if promptViaPaneText(agent) {
			t.Errorf("promptViaPaneText(%q) = true, want false", agent)
		}
	}
}
