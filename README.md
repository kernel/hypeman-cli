# Hypeman CLI

The official CLI for the Hypeman REST API.

It is generated with [Stainless](https://www.stainless.com/).

<!-- x-release-please-start-version -->

## Installation

### Installing with Homebrew

```sh
brew tap kernel/tap
brew install hypeman
```

### Installing with Go

```sh
go install 'github.com/kernel/hypeman-cli/cmd/hypeman@latest'
```

<!-- x-release-please-end -->

### Running Locally

```sh
./scripts/run args...
```

## Usage

The CLI follows a resource-based command structure:

```sh
hypeman [resource] [command] [flags]
```

```sh
hypeman health check
```

For details about specific commands, use the `--help` flag.

## Global Flags

- `--help` - Show command line usage
- `--debug` - Enable debug logging (includes HTTP request/response details)
- `--version`, `-v` - Show the CLI version
- `--base-url` - Use a custom API backend URL
- `--format` - Change the output format (`auto`, `explore`, `json`, `jsonl`, `pretty`, `raw`, `yaml`)
- `--format-error` - Change the output format for errors (`auto`, `explore`, `json`, `jsonl`, `pretty`, `raw`, `yaml`)
- `--transform` - Transform the data output using [GJSON syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)
- `--transform-error` - Transform the error output using [GJSON syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)
