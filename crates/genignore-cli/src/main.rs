//! genignore CLI entry point — usage-rs driven grammar with cobra-compatible
//! help, usage, and error output.

mod help;
mod output;
mod spec;

use std::io::Write;
use std::process::ExitCode;

use genignore_core::{catalog_result, load_config, Service};
use usage::parse::ParseValue;

fn main() -> ExitCode {
    // Match Go's default SIGPIPE behavior: die silently on closed stdout.
    #[cfg(unix)]
    unsafe {
        libc::signal(libc::SIGPIPE, libc::SIG_DFL);
    }
    let args: Vec<String> = std::env::args().skip(1).collect();
    match run(&args) {
        Ok(()) => ExitCode::SUCCESS,
        Err(code) => code,
    }
}

/// First token that doesn't start with `-` (flag values of `--include` /
/// `--exclude` are skipped since those take values).
fn command_word(args: &[String]) -> Option<&str> {
    let mut i = 0;
    while i < args.len() {
        let tok = &args[i];
        if tok == "--include" || tok == "--exclude" {
            i += 2;
            continue;
        }
        if !tok.starts_with('-') {
            return Some(tok);
        }
        i += 1;
    }
    None
}

fn has_help_flag(args: &[String]) -> bool {
    args.iter().any(|t| t == "-h" || t == "--help")
}

const KNOWN_COMMANDS: [&str; 6] = ["add", "detect", "doctor", "list", "resolve", "search"];

fn run(args: &[String]) -> Result<(), ExitCode> {
    // `help` pseudo-command.
    if command_word(args) == Some("help") {
        let mut it = args.iter().skip_while(|t| *t != "help").skip(1);
        if has_help_flag(args) {
            print!("{}", help::command_help("help").unwrap());
            return Ok(());
        }
        let topic = it.find(|t| !t.starts_with('-')).map(|s| s.as_str());
        return match topic {
            None => {
                print!("{}", help::ROOT_HELP);
                Ok(())
            }
            Some(cmd) => match help::command_help(cmd) {
                Some(text) => {
                    print!("{}", text);
                    Ok(())
                }
                None => {
                    eprintln!("Unknown help topic [`{}`]", cmd);
                    eprint!("{}", help::help_topic_usage());
                    Ok(())
                }
            },
        };
    }

    // `completion` pseudo-command.
    if command_word(args) == Some("completion") {
        let mut it = args
            .iter()
            .skip_while(|t| *t != "completion")
            .skip(1)
            .filter(|t| !t.starts_with('-'));
        let shell = it.next().map(|s| s.as_str());
        return match shell {
            None => {
                print!("{}", help::completion_help());
                Ok(())
            }
            Some("bash" | "zsh" | "fish" | "powershell") => {
                match usage::complete::complete(&usage::complete::CompleteOptions {
                    usage_bin: "genignore".to_string(),
                    shell: shell.unwrap().to_string(),
                    bin: "genignore".to_string(),
                    cache_key: None,
                    spec: Some(spec::spec().clone()),
                    usage_cmd: None,
                    source_file: None,
                }) {
                    Ok(script) => {
                        println!("{}", script);
                        Ok(())
                    }
                    Err(e) => {
                        eprintln!("error: {}", e);
                        Err(ExitCode::FAILURE)
                    }
                }
            }
            Some(_) => {
                print!("{}", help::completion_help());
                Ok(())
            }
        };
    }

    // usage-rs parse over the spec grammar.
    let mut full_args = vec!["genignore".to_string()];
    full_args.extend(args.iter().cloned());
    let out = match usage::parse(spec::spec(), &full_args) {
        Ok(out) => out,
        Err(err) => {
            let msg = err.to_string();
            if has_help_flag(args) {
                // cobra checks the help flag before required-arg validation;
                // an unknown flag still errors first.
                if msg.starts_with("Missing required arg:")
                    || msg.starts_with("Missing required flag:")
                {
                    let cmd = command_word(args).unwrap_or("");
                    print!("{}", help::command_help(cmd).unwrap_or(help::ROOT_HELP));
                    return Ok(());
                }
                // trailing positionals + -h: cobra prints root help.
                if msg.starts_with("unexpected word: ") {
                    let word = msg.trim_start_matches("unexpected word: ");
                    if !word.starts_with('-') {
                        print!("{}", help::ROOT_HELP);
                        return Ok(());
                    }
                }
            }
            return Err(flag_error(args, &msg));
        }
    };

    let cmd_name = out.cmd.name.clone();
    let cmd = if KNOWN_COMMANDS.contains(&cmd_name.as_str()) {
        cmd_name.as_str()
    } else {
        ""
    };

    let flag_bool = |name: &str| {
        out.flags
            .iter()
            .any(|(f, v)| f.name == name && matches!(v, ParseValue::Bool(true)))
    };
    if flag_bool("help") {
        print!("{}", help::command_help(cmd).unwrap_or(help::ROOT_HELP));
        return Ok(());
    }

    // usage-lib var-args greedily absorb `--x` tokens into positional values;
    // cobra would reject them as unknown flags. Values after `--` stay
    // positional.
    let after_ddash: Vec<&str> = {
        let mut v = Vec::new();
        let mut seen = false;
        for tok in args {
            if seen {
                v.push(tok.as_str());
            } else if tok == "--" {
                seen = true;
            }
        }
        v
    };
    let positional_values = |name: &str| -> Vec<String> {
        out.args
            .iter()
            .filter(|(a, _)| a.name == name)
            .flat_map(|(_, v)| match v {
                ParseValue::MultiString(vs) => vs.clone(),
                ParseValue::String(s) => vec![s.clone()],
                _ => Vec::new(),
            })
            .collect()
    };
    for value in positional_values("rest")
        .into_iter()
        .chain(positional_values("providers"))
    {
        if value.starts_with('-') && value != "-" && !after_ddash.contains(&value.as_str()) {
            let msg = if value.starts_with("--") {
                format!(
                    "unknown flag: {}",
                    value.split('=').next().unwrap_or(&value)
                )
            } else {
                format!("unknown shorthand flag: '{}' in {}", &value[1..2], value)
            };
            return Err(command_error(cmd, &msg));
        }
    }

    if cmd.is_empty() {
        print!("{}", help::ROOT_HELP);
        return Ok(());
    }

    let flag_values = |name: &str| -> Vec<String> {
        out.flags
            .iter()
            .filter(|(f, _)| f.name == name)
            .flat_map(|(_, v)| {
                v.try_as_multi_string_ref()
                    .cloned()
                    .unwrap_or_default()
                    .into_iter()
            })
            .flat_map(|s| s.split(',').map(|p| p.to_string()).collect::<Vec<_>>())
            .collect()
    };
    let arg_values = |name: &str| -> Vec<String> {
        out.args
            .iter()
            .filter(|(a, _)| a.name == name)
            .flat_map(|(_, v)| match v {
                ParseValue::MultiString(vs) => vs.clone(),
                ParseValue::String(s) => vec![s.clone()],
                _ => Vec::new(),
            })
            .collect()
    };

    let json_output = flag_bool("json");
    let verbose = flag_bool("verbose");
    let dry_run = flag_bool("dry-run");
    let diff = flag_bool("diff");

    // search exact-arg enforcement (cobra ExactArgs(1)).
    if cmd == "search" {
        let rest = arg_values("rest").len();
        if rest > 0 {
            return Err(command_error(
                "search",
                &format!("accepts 1 arg(s), received {}", rest + 1),
            ));
        }
    }

    let cwd = match std::env::current_dir() {
        Ok(c) => c,
        Err(e) => {
            eprintln!("error: resolve working directory: {}", e);
            return Err(ExitCode::FAILURE);
        }
    };
    let config = match load_config() {
        Ok(c) => c,
        Err(e) => {
            eprintln!("error: {}", e);
            return Err(ExitCode::FAILURE);
        }
    };

    match cmd {
        "list" | "search" => {
            let query = if cmd == "search" {
                arg_values("term").into_iter().next().unwrap_or_default()
            } else {
                String::new()
            };
            match catalog_result(cmd, &query) {
                Ok(res) => {
                    output::print_catalog_result(&res, json_output);
                    Ok(())
                }
                Err(e) => Err(command_error(cmd, &e)),
            }
        }
        _ => {
            let service = match Service::new(cwd, config) {
                Ok(s) => s,
                Err(e) => {
                    eprintln!("error: {}", e);
                    return Err(ExitCode::FAILURE);
                }
            };
            let include = flag_values("include");
            let exclude = flag_values("exclude");
            match cmd {
                "resolve" => match service.resolve(&include, &exclude) {
                    Ok(res) => {
                        output::print_resolve_result(&res, json_output, verbose);
                        Ok(())
                    }
                    Err(e) => Err(command_error(cmd, &e)),
                },
                "detect" => match service.detect(&include, &exclude, dry_run, diff) {
                    Ok(res) => {
                        output::print_result(&res, json_output, verbose);
                        Ok(())
                    }
                    Err(e) => Err(command_error(cmd, &e)),
                },
                "add" => {
                    let keys = arg_values("providers");
                    match service.add(&keys, dry_run, diff) {
                        Ok(res) => {
                            output::print_result(&res, json_output, verbose);
                            Ok(())
                        }
                        Err(e) => Err(command_error(cmd, &e)),
                    }
                }
                "doctor" => match service.doctor(&include, &exclude) {
                    Ok(res) => {
                        output::print_doctor_result(&res, json_output);
                        Ok(())
                    }
                    Err(e) => Err(command_error(cmd, &e)),
                },
                _ => unreachable!(),
            }
        }
    }
}

/// cobra `Error: <msg>` + command usage block + `error: <msg>`, exit 1.
fn command_error(cmd: &str, msg: &str) -> ExitCode {
    let mut stderr = std::io::stderr().lock();
    let _ = writeln!(stderr, "Error: {}", msg);
    let _ = write!(stderr, "{}", help::command_usage(cmd));
    let _ = writeln!(stderr, "\nerror: {}", msg);
    ExitCode::FAILURE
}

/// Map a usage-rs parse error onto the cobra-equivalent message + usage.
fn flag_error(args: &[String], msg: &str) -> ExitCode {
    let cmd = match command_word(args) {
        Some(w) if KNOWN_COMMANDS.contains(&w) => w,
        _ => "",
    };
    if let Some(word) = msg.strip_prefix("unexpected word: ") {
        if word.starts_with('-') {
            return command_error(cmd, &format!("unknown flag: {}", word));
        }
        // Unknown command: no usage block, just the help hint.
        let mut stderr = std::io::stderr().lock();
        let _ = writeln!(
            stderr,
            "Error: unknown command {:?} for \"genignore\"",
            word
        );
        let _ = writeln!(stderr, "Run 'genignore --help' for usage.");
        let _ = writeln!(
            stderr,
            "error: unknown command {:?} for \"genignore\"",
            word
        );
        return ExitCode::FAILURE;
    }
    if msg.contains("requires an argument") {
        // `Invalid flag `--x`: requires an argument` → cobra wording.
        if let Some(start) = msg.find('`') {
            if let Some(end) = msg[start + 1..].find('`') {
                let flag = &msg[start + 1..start + 1 + end];
                return command_error(cmd, &format!("flag needs an argument: {}", flag));
            }
        }
    }
    if let Some(rest) = msg.strip_prefix("Missing required arg: <") {
        let name = rest.trim_end_matches('>');
        let text = match name {
            "providers" => "requires at least 1 arg(s), only received 0".to_string(),
            "term" => "accepts 1 arg(s), received 0".to_string(),
            other => format!("missing required argument: {}", other),
        };
        return command_error(cmd, &text);
    }
    command_error(cmd, msg)
}
