package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// promptLine reads a trimmed line from stdin.
func promptLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// promptInput reads a non-empty trimmed line.
func promptInput(reader *bufio.Reader, prompt string) (string, error) {
	for {
		line, err := promptLine(reader, prompt)
		if err != nil {
			return "", err
		}
		if line != "" {
			return line, nil
		}
	}
}

// promptSecret reads a masked line.
func promptSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// promptSelect shows a numbered menu and returns the chosen index (0-based).
func promptSelect(reader *bufio.Reader, title string, options []string) (int, error) {
	fmt.Println()
	fmt.Println("  " + title)
	fmt.Println()
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Println()
	for {
		input, err := promptInput(reader, "  Select [number]: ")
		if err != nil {
			return 0, err
		}
		idx, err := strconv.Atoi(input)
		if err == nil && idx >= 1 && idx <= len(options) {
			return idx - 1, nil
		}
		fmt.Printf("  invalid selection: %s\n", input)
	}
}

// promptYesNo asks a yes/no question, defaulting to def.
func promptYesNo(reader *bufio.Reader, prompt string, def bool) (bool, error) {
	suffix := " [y/N]: "
	if def {
		suffix = " [Y/n]: "
	}
	for {
		input, err := promptLine(reader, prompt+suffix)
		if err != nil {
			return def, err
		}
		switch strings.ToLower(input) {
		case "":
			return def, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Printf("  please answer y or n\n")
	}
}
