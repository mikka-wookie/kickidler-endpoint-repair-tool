package wizard

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Prompter interface {
	Confirm(message string, defaultValue bool) (bool, error)
	AskString(message string, allowEmpty bool) (string, error)
	AskSecret(message string, allowEmpty bool) (string, error)
}

type ConsolePrompter struct {
	In  io.Reader
	Out io.Writer
}

func (p ConsolePrompter) Confirm(message string, defaultValue bool) (bool, error) {
	suffix := " [y/N]: "
	if defaultValue {
		suffix = " [Y/n]: "
	}
	answer, err := p.askLine(message+suffix, true)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(answer) == "" {
		return defaultValue, nil
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (p ConsolePrompter) AskString(message string, allowEmpty bool) (string, error) {
	return p.askLine(message+": ", allowEmpty)
}

func (p ConsolePrompter) AskSecret(message string, allowEmpty bool) (string, error) {
	if p.Out != nil {
		fmt.Fprintln(p.Out, "Invite input may be visible in this console. It will not be written to reports.")
	}
	return p.askLine(message+": ", allowEmpty)
}

func (p ConsolePrompter) askLine(message string, allowEmpty bool) (string, error) {
	if p.Out != nil {
		fmt.Fprint(p.Out, message)
	}
	reader := bufio.NewReader(p.In)
	for {
		value, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		value = strings.TrimRight(value, "\r\n")
		if allowEmpty || strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
		if p.Out != nil {
			fmt.Fprint(p.Out, message)
		}
		if err == io.EOF {
			return "", nil
		}
	}
}

type FakePrompter struct {
	Confirms []bool
	Strings  []string
	Secrets  []string
	Asked    []string
}

func (p *FakePrompter) Confirm(message string, defaultValue bool) (bool, error) {
	p.Asked = append(p.Asked, message)
	if len(p.Confirms) == 0 {
		return defaultValue, nil
	}
	value := p.Confirms[0]
	p.Confirms = p.Confirms[1:]
	return value, nil
}

func (p *FakePrompter) AskString(message string, allowEmpty bool) (string, error) {
	p.Asked = append(p.Asked, message)
	if len(p.Strings) == 0 {
		return "", nil
	}
	value := p.Strings[0]
	p.Strings = p.Strings[1:]
	return value, nil
}

func (p *FakePrompter) AskSecret(message string, allowEmpty bool) (string, error) {
	p.Asked = append(p.Asked, message)
	if len(p.Secrets) == 0 {
		return "", nil
	}
	value := p.Secrets[0]
	p.Secrets = p.Secrets[1:]
	return value, nil
}
