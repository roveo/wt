package shell

import "fmt"

// BashInit returns the bash initialization script
func BashInit() string {
	return `# wt shell integration for bash
wt() {
    local result exit_code cd_cmd
    result="$(command wt "$@")"
    exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        [[ -n "$result" ]] && echo "$result" >&2
        return $exit_code
    fi
    # Extract cd command (handles any escape sequences that might be present)
    cd_cmd=$(echo "$result" | sed -n 's/.*\(cd "[^"]*"\).*/\1/p' | tail -1)
    if [[ -n "$cd_cmd" ]]; then
        eval "$cd_cmd"
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
    local result exit_code cd_cmd
    result="$(command wt "$@")"
    exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        [[ -n "$result" ]] && echo "$result" >&2
        return $exit_code
    fi
    # Extract cd command (handles any escape sequences that might be present)
    cd_cmd=$(echo "$result" | sed -n 's/.*\(cd "[^"]*"\).*/\1/p' | tail -1)
    if [[ -n "$cd_cmd" ]]; then
        eval "$cd_cmd"
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
    # Get the cd command line
    set -l cd_cmd (echo "$result" | grep -E '^cd ' | tail -1)
    if test -n "$cd_cmd"
        eval $cd_cmd
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
