hpf() {
    os=$(uname -s)

    # Linux
    if [[ "$os" == "Linux" ]]; then
        export HPF_LAST_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/hyperfile/lastdir"
    fi

    # macOS
    if [[ "$os" == "Darwin" ]]; then
        export HPF_LAST_DIR="$HOME/Library/Application Support/hyperfile/lastdir"
    fi

    command hpf "$@"

    [ ! -f "$HPF_LAST_DIR" ] || {
        . "$HPF_LAST_DIR"
        rm -f -- "$HPF_LAST_DIR" > /dev/null
    }
}
