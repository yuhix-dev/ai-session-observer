package viewer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stdout.Len() > 0 {
			return stdout.String(), nil
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

func currentPanePID(panePID, paneTarget string) string {
	if paneTarget == "" {
		return panePID
	}
	out, err := runCommand("env", "-u", "TMUX", "tmux", "display-message", "-p", "-t", paneTarget, "#{pane_pid}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func processTree(root string) []int {
	rootPID, err := strconv.Atoi(strings.TrimSpace(root))
	if err != nil || rootPID <= 0 {
		return nil
	}
	if err := syscallKillZero(rootPID); err != nil {
		return nil
	}

	out, err := runCommand("ps", "-ax", "-o", "pid=,ppid=")
	if err != nil {
		return nil
	}
	children := map[int][]int{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}

	stack := []int{rootPID}
	seen := map[int]bool{}
	var pids []int
	for len(stack) > 0 {
		pid := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		pids = append(pids, pid)
		stack = append(stack, children[pid]...)
	}
	sort.Ints(pids)
	return pids
}

func syscallKillZero(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.Signal(0))
}

func lsofRolloutFiles(pids []int) []string {
	if len(pids) == 0 {
		return nil
	}
	items := make([]string, 0, len(pids))
	for _, pid := range pids {
		items = append(items, strconv.Itoa(pid))
	}
	out, err := runCommand("lsof", "-p", strings.Join(items, ","))
	if err != nil && out == "" {
		return nil
	}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		path := fields[len(fields)-1]
		if strings.HasSuffix(path, ".jsonl") && strings.Contains(filepath.Base(path), "rollout-") {
			seen[path] = true
		}
	}
	return sortedKeys(seen)
}

func discoverCodexFiles(opts Options) []string {
	if pid := currentPanePID(opts.PanePID, opts.PaneTarget); pid != "" {
		if files := lsofRolloutFiles(processTree(pid)); len(files) > 0 {
			return files
		}
	}
	root := filepath.Join(os.Getenv("HOME"), ".codex", "sessions")
	return latestMatchingFile(root, "rollout-*.jsonl")
}

func discoverClaudeFiles(opts Options) []string {
	if pid := currentPanePID(opts.PanePID, opts.PaneTarget); pid != "" {
		if files := paneClaudeFiles(pid); len(files) > 0 {
			return files
		}
	}
	root := filepath.Join(os.Getenv("HOME"), ".claude", "projects")
	return latestMatchingFile(root, "*.jsonl")
}

func paneClaudeFiles(panePID string) []string {
	root := filepath.Join(os.Getenv("HOME"), ".claude")
	sessionDir := filepath.Join(root, "sessions")
	projectDir := filepath.Join(root, "projects")
	sessionIDs := map[string]bool{}
	for _, pid := range processTree(panePID) {
		path := filepath.Join(sessionDir, strconv.Itoa(pid)+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}
		if sessionID := stringValue(payload["sessionId"]); sessionID != "" {
			sessionIDs[sessionID] = true
		}
	}

	var files []string
	for sessionID := range sessionIDs {
		_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if filepath.Base(path) == sessionID+".jsonl" {
				files = append(files, path)
			}
			return nil
		})
	}
	sort.Strings(files)
	return uniqueStrings(files)
}

func latestMatchingFile(root, pattern string) []string {
	type match struct {
		path string
		mod  time.Time
	}
	var matches []match
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ok, globErr := filepath.Match(pattern, filepath.Base(path))
		if globErr != nil || !ok {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		matches = append(matches, match{path: path, mod: info.ModTime()})
		return nil
	})
	if len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].mod.Before(matches[j].mod)
	})
	return []string{matches[len(matches)-1].path}
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	var uniq []string
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		uniq = append(uniq, item)
	}
	sort.Strings(uniq)
	return uniq
}

func sortedKeys(items map[string]bool) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func readTailLines(path string, limit int) ([]string, error) {
	handle, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	var lines []string
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(lines) <= limit {
		return lines, nil
	}
	return lines[len(lines)-limit:], nil
}
