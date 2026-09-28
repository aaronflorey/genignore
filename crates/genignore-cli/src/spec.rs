//! usage-lib spec describing the genignore command grammar.

pub const SPEC: &str = r#"
bin "genignore"
flag "--json" global=#true
flag "--verbose" global=#true
flag "-h --help" global=#true
cmd "add" {
    flag "--diff"
    flag "--dry-run"
    arg "providers" var=#true
}
cmd "detect" {
    flag "--diff"
    flag "--dry-run"
    flag "--exclude <keys>" var=#true
    flag "--include <keys>" var=#true
    arg "rest" var=#true required=#false
}
cmd "doctor" {
    flag "--exclude <keys>" var=#true
    flag "--include <keys>" var=#true
    arg "rest" var=#true required=#false
}
cmd "list" {
    arg "rest" var=#true required=#false
}
cmd "resolve" {
    flag "--exclude <keys>" var=#true
    flag "--include <keys>" var=#true
    arg "rest" var=#true required=#false
}
cmd "search" {
    arg "term"
    arg "rest" var=#true required=#false
}
"#;

pub fn spec() -> &'static usage::Spec {
    static SPEC_LOCK: std::sync::OnceLock<usage::Spec> = std::sync::OnceLock::new();
    SPEC_LOCK.get_or_init(|| SPEC.parse::<usage::Spec>().expect("invalid usage spec"))
}

#[cfg(test)]
mod tests {
    use super::spec;
    use usage::parse::{ParseOutput, ParseValue};

    fn parse(args: &[&str]) -> Result<ParseOutput, usage::miette::Error> {
        let mut full = vec!["genignore".to_string()];
        full.extend(args.iter().map(|s| s.to_string()));
        usage::parse(spec(), &full)
    }

    fn flag_bool(out: &ParseOutput, name: &str) -> bool {
        out.flags
            .iter()
            .any(|(f, v)| f.name == name && matches!(v, ParseValue::Bool(true)))
    }

    fn flag_values(out: &ParseOutput, name: &str) -> Vec<String> {
        out.flags
            .iter()
            .filter(|(f, _)| f.name == name)
            .flat_map(|(_, v)| v.try_as_multi_string_ref().cloned().unwrap_or_default())
            .collect()
    }

    fn arg_values(out: &ParseOutput, name: &str) -> Vec<String> {
        out.args
            .iter()
            .filter(|(a, _)| a.name == name)
            .flat_map(|(_, v)| match v {
                ParseValue::MultiString(vs) => vs.clone(),
                ParseValue::String(s) => vec![s.clone()],
                _ => Vec::new(),
            })
            .collect()
    }

    #[test]
    fn parses_all_commands() {
        for cmd in ["add", "detect", "doctor", "list", "resolve", "search"] {
            let args: &[&str] = if cmd == "add" {
                &[cmd, "node"]
            } else if cmd == "search" {
                &[cmd, "term"]
            } else {
                &[cmd]
            };
            let out = parse(args).unwrap_or_else(|e| panic!("{cmd}: {e}"));
            assert_eq!(out.cmd.name, cmd);
        }
    }

    #[test]
    fn global_flags_work_everywhere() {
        for cmd in ["list", "detect", "doctor", "resolve"] {
            let out = parse(&[cmd, "--json", "--verbose"]).unwrap();
            assert!(flag_bool(&out, "json"));
            assert!(flag_bool(&out, "verbose"));
        }
        let out = parse(&["--json", "list"]).unwrap();
        assert!(flag_bool(&out, "json"));
        assert_eq!(out.cmd.name, "list");
    }

    #[test]
    fn include_exclude_repeatable() {
        let out = parse(&[
            "detect",
            "--include",
            "node,go",
            "--include",
            "rust",
            "--exclude",
            "java",
        ])
        .unwrap();
        assert_eq!(
            flag_values(&out, "include"),
            vec!["node,go".to_string(), "rust".to_string()]
        );
        assert_eq!(flag_values(&out, "exclude"), vec!["java".to_string()]);
    }

    #[test]
    fn add_takes_provider_args() {
        let out = parse(&["add", "node", "rust", "--dry-run"]).unwrap();
        assert_eq!(out.cmd.name, "add");
        assert!(flag_bool(&out, "dry-run"));
        assert_eq!(
            arg_values(&out, "providers"),
            vec!["node".to_string(), "rust".to_string()]
        );
    }

    #[test]
    fn detect_and_add_accept_diff_and_dry_run() {
        let out = parse(&["detect", "--diff", "--dry-run"]).unwrap();
        assert!(flag_bool(&out, "diff"));
        assert!(flag_bool(&out, "dry-run"));
        let out = parse(&["add", "node", "--diff"]).unwrap();
        assert!(flag_bool(&out, "diff"));
    }

    #[test]
    fn search_requires_term() {
        assert!(parse(&["search"]).is_err());
        assert!(parse(&["search", "rust"]).is_ok());
        let out = parse(&["search", "rust", "extra"]).unwrap();
        assert_eq!(arg_values(&out, "rest"), vec!["extra".to_string()]);
    }

    #[test]
    fn unknown_command_is_error() {
        assert!(parse(&["bogus"]).is_err());
    }

    #[test]
    fn unknown_flag_lands_in_var_args() {
        // usage-lib var-args absorb `--x` tokens positionally; main.rs's
        // post-parse leading-dash check turns them into cobra's
        // `unknown flag` errors. `--` keeps later values positional.
        let out = parse(&["list", "--nope"]).unwrap();
        assert_eq!(arg_values(&out, "rest"), vec!["--nope".to_string()]);
    }

    #[test]
    fn help_flag_parses() {
        let out = parse(&["detect", "-h"]).unwrap();
        assert!(flag_bool(&out, "help"));
        let out = parse(&["--help"]).unwrap();
        assert!(flag_bool(&out, "help"));
    }
}
