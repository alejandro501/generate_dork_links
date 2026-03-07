# generate_dork_links

Go CLI for generating GitHub, Google, Shodan, and Wayback dork links from reusable wordlists.

## Build

```sh
go build -o generate_dork_links
```

The binary is intended to be installed (via `go install` or a package manager) and run like other penetration‑testing tools such as `ffuf`. You can also execute it with `go run main.go` during development.

## Usage

```
generate_dork_links --organization example.com --all
```

Supported flags:

| Flag | Description |
| --- | --- |
| `--organization` | Specify a single organization to target. |
| `--list` | Path to a newline separated file with organizations. |
| `--output` | Word to prefix generated filenames inside `./dorking/<type>/`. |
| `--output-github`, `--output-google`, `--output-shodan`, `--output-wayback` | Override the destination file for each product. |
| `-w`, `--wordlist` | Wordlist file used for every enabled dork type (like `ffuf -w`). |
| `--wordlist-github` (`-wGh`), `--wordlist-google` (`-wGg`), `--wordlist-shodan` (`-wSh`), `--wordlist-wayback` (`-wWb`) | Override the wordlist for a specific dork type. |
| `--github`, `--google`, `--shodan`, `--wayback`, `--all` | Enable specific dork categories (`--all` runs all of them and is the default when no category flags are set). |

### Environment variables

| Variable | Purpose |
| --- | --- |
| `TARGET` | Base directory used when `DORKING` is unspecified. Defaults to the current working directory. |
| `DORKING` | Base directory where generated links are written by type (for example, `${DORKING}/github/...`). Falls back to `${TARGET}/dorking`. |

## Wordlists

The tool does not assume any embedded wordlists. You must supply one using `-w/--wordlist`, and you can override specific dork categories via the `--wordlist-*` flags (seen above).

## Dork Formats

The generated links use the following formats.

| Service | Format |
| --- | --- |
| GitHub | `"<word>" "<organization>"` |
| Google | `"<word>" site:<organization>` |
| Shodan | `"<word>" hostname:"<organization>"` |
| Wayback | `<organization> <word>` |
