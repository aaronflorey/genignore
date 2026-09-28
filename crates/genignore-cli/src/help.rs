//! Cobra-compatible help and usage text, captured verbatim from the Go CLI.

pub const ROOT_HELP: &str = r#"Generate and manage gitignore block

Usage:
  genignore [command]

Available Commands:
  add         Add providers to existing managed set
  completion  Generate the autocompletion script for the specified shell
  detect      Detect providers and rebuild managed block
  doctor      Explain detection, provider resolution, and runtime decisions
  help        Help about any command
  list        List all supported provider keys
  resolve     Resolve providers without mutating .gitignore
  search      Search supported provider keys

Flags:
  -h, --help      help for genignore
      --json      output machine-readable JSON
      --verbose   show verbose detection info

Use "genignore [command] --help" for more information about a command.
"#;

const ADD_HELP: &str = r#"Add providers to existing managed set

Usage:
  genignore add <keys...> [flags]

Flags:
      --diff      show the exact managed-block diff without writing files
      --dry-run   show what would change without writing files
  -h, --help      help for add

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info
"#;

const DETECT_HELP: &str = r#"Detect providers and rebuild managed block

Usage:
  genignore detect [flags]

Flags:
      --diff              show the exact managed-block diff without writing files
      --dry-run           show what would change without writing files
      --exclude strings   provider keys to exclude
  -h, --help              help for detect
      --include strings   provider keys to include

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info
"#;

const DOCTOR_HELP: &str = r#"Explain detection, provider resolution, and runtime decisions

Usage:
  genignore doctor [flags]

Flags:
      --exclude strings   provider keys to exclude
  -h, --help              help for doctor
      --include strings   provider keys to include

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info
"#;

const LIST_HELP: &str = r#"List all supported provider keys

Usage:
  genignore list [flags]

Flags:
  -h, --help   help for list

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info
"#;

const RESOLVE_HELP: &str = r#"Resolve providers without mutating .gitignore

Usage:
  genignore resolve [flags]

Flags:
      --exclude strings   provider keys to exclude
  -h, --help              help for resolve
      --include strings   provider keys to include

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info
"#;

const SEARCH_HELP: &str = r#"Search supported provider keys

Usage:
  genignore search <term> [flags]

Flags:
  -h, --help   help for search

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info
"#;

const HELP_HELP: &str = r#"Help provides help for any command in the application.
Simply type genignore help [path to command] for full details.

Usage:
  genignore help [command] [flags]

Flags:
  -h, --help   help for help

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info
"#;

const COMPLETION_HELP: &str = r#"Generate the autocompletion script for genignore for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  genignore completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Global Flags:
      --json      output machine-readable JSON
      --verbose   show verbose detection info

Use "genignore completion [command] --help" for more information about a command.
"#;

pub fn command_help(cmd: &str) -> Option<&'static str> {
    match cmd {
        "add" => Some(ADD_HELP),
        "detect" => Some(DETECT_HELP),
        "doctor" => Some(DOCTOR_HELP),
        "list" => Some(LIST_HELP),
        "resolve" => Some(RESOLVE_HELP),
        "search" => Some(SEARCH_HELP),
        "help" => Some(HELP_HELP),
        "completion" => Some(COMPLETION_HELP),
        _ => None,
    }
}

/// The usage block cobra prints on errors: the command help without its
/// first description line.
pub fn command_usage(cmd: &str) -> &'static str {
    let help = match cmd {
        "" => ROOT_HELP,
        other => command_help(other).unwrap_or(ROOT_HELP),
    };
    // Strip the first line and the following blank line (the Short text).
    match help.find('\n') {
        Some(idx) => {
            let rest = &help[idx + 1..];
            rest.strip_prefix('\n').unwrap_or(rest)
        }
        None => help,
    }
}

/// Usage block for the `help <topic>` failure path: root usage with the
/// `-h, --help` flag line removed (matches cobra's help command output).
pub fn help_topic_usage() -> &'static str {
    "Usage:\n  genignore [command]\n\nAvailable Commands:\n  add         Add providers to existing managed set\n  completion  Generate the autocompletion script for the specified shell\n  detect      Detect providers and rebuild managed block\n  doctor      Explain detection, provider resolution, and runtime decisions\n  help        Help about any command\n  list        List all supported provider keys\n  resolve     Resolve providers without mutating .gitignore\n  search      Search supported provider keys\n\nFlags:\n      --json      output machine-readable JSON\n      --verbose   show verbose detection info\n\nUse \"genignore [command] --help\" for more information about a command.\n"
}

/// The full completion command help (cobra prints it whole, description line
/// included, when the command runs bare or with an unknown shell).
pub fn completion_help() -> &'static str {
    COMPLETION_HELP
}
