package shell

import "fmt"

// BashInit returns the bash initialization script
func BashInit() string {
	return `# wt shell integration for bash
wt() {
    local result exit_code first_line
    result="$(command wt "$@")"
    exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        [[ -n "$result" ]] && echo "$result" >&2
        return $exit_code
    fi
    # Check if first line is a cd command
    first_line="${result%%$'\n'*}"
    if [[ "$first_line" == cd\ * ]]; then
        eval "$result"
    elif [[ -n "$result" ]]; then
        echo "$result"
    fi
}
`
}

// ZshInit returns the zsh initialization script
func ZshInit() string {
	return `# wt shell integration for zsh
wt() {
    local result exit_code first_line
    result="$(command wt "$@")"
    exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        [[ -n "$result" ]] && echo "$result" >&2
        return $exit_code
    fi
    # Check if first line is a cd command
    first_line="${result%%$'\n'*}"
    if [[ "$first_line" == cd\ * ]]; then
        eval "$result"
    elif [[ -n "$result" ]]; then
        echo "$result"
    fi
}
`
}

// FishInit returns the fish initialization script
func FishInit() string {
	return `# wt shell integration for fish
function wt
    set -l result (command wt $argv)
    set -l exit_code $status
    if test $exit_code -ne 0
        echo $result >&2
        return $exit_code
    end
    # Check if first line is a cd command
    set -l first_line $result[1]
    if string match -q 'cd *' "$first_line"
        eval (string join "; " $result)
    else if test -n "$result"
        echo $result
    end
end
`
}

// GetInit returns the initialization script for the given shell
func GetInit(shell string) (string, error) {
	switch shell {
	case "bash":
		return BashInit(), nil
	case "zsh":
		return ZshInit(), nil
	case "fish":
		return FishInit(), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
}
