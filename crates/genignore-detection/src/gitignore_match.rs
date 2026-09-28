//! Port of go-git's `plumbing/format/gitignore` pattern parsing and matching,
//! used to decide which one-level subdirectories detection may descend into.
//! Detection only ever matches depth-1 names, so only root-domain patterns
//! (root `.gitignore` plus root `.git/info/exclude`) can apply.

use std::fs;
use std::io;
use std::path::Path;

use crate::globmatch::path_match;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum MatchResult {
    NoMatch,
    Exclude,
    Include,
}

struct Pattern {
    domain: Vec<String>,
    pattern: Vec<String>,
    inclusion: bool,
    dir_only: bool,
    is_glob: bool,
}

impl Pattern {
    fn parse(mut p: &str, domain: &[String]) -> Pattern {
        let mut inclusion = false;
        if let Some(rest) = p.strip_prefix('!') {
            inclusion = true;
            p = rest;
        }
        let mut p = if !p.ends_with("\\ ") {
            p.trim_end_matches(' ').to_string()
        } else {
            p.to_string()
        };
        let mut dir_only = false;
        if p.ends_with('/') {
            dir_only = true;
            p.pop();
        }
        let is_glob = p.contains('/');
        Pattern {
            domain: domain.to_vec(),
            pattern: p.split('/').map(|s| s.to_string()).collect(),
            inclusion,
            dir_only,
            is_glob,
        }
    }

    fn matches(&self, path: &[String], is_dir: bool) -> MatchResult {
        if path.len() <= self.domain.len() {
            return MatchResult::NoMatch;
        }
        for (i, e) in self.domain.iter().enumerate() {
            if &path[i] != e {
                return MatchResult::NoMatch;
            }
        }
        let path = &path[self.domain.len()..];
        let ok = if self.is_glob {
            self.glob_match(path, is_dir)
        } else {
            self.simple_name_match(path, is_dir)
        };
        if !ok {
            return MatchResult::NoMatch;
        }
        if self.inclusion {
            MatchResult::Include
        } else {
            MatchResult::Exclude
        }
    }

    fn simple_name_match(&self, path: &[String], is_dir: bool) -> bool {
        for (i, name) in path.iter().enumerate() {
            match path_match(&self.pattern[0], name) {
                Err(_) => return false,
                Ok(false) => continue,
                Ok(true) => {
                    if self.dir_only && !is_dir && i == path.len() - 1 {
                        return false;
                    }
                    return true;
                }
            }
        }
        false
    }

    fn glob_match(&self, path: &[String], is_dir: bool) -> bool {
        let mut path: &[String] = path;
        let mut matched = false;
        let mut can_traverse = false;
        for (i, pattern) in self.pattern.iter().enumerate() {
            if pattern.is_empty() {
                can_traverse = false;
                continue;
            }
            if pattern == "**" {
                if i == self.pattern.len() - 1 {
                    break;
                }
                can_traverse = true;
                continue;
            }
            if pattern.contains("**") {
                return false;
            }
            if path.is_empty() {
                return false;
            }
            if can_traverse {
                can_traverse = false;
                loop {
                    if path.is_empty() {
                        matched = false;
                        break;
                    }
                    let e = path[0].clone();
                    path = &path[1..];
                    match path_match(pattern, &e) {
                        Err(_) => return false,
                        Ok(true) => {
                            matched = true;
                            break;
                        }
                        Ok(false) => {}
                    }
                }
            } else {
                match path_match(pattern, &path[0]) {
                    Err(_) | Ok(false) => return false,
                    Ok(true) => {
                        matched = true;
                        path = &path[1..];
                    }
                }
            }
        }
        if matched && self.dir_only && !is_dir && path.is_empty() {
            matched = false;
        }
        matched
    }
}

pub struct Matcher {
    patterns: Vec<Pattern>,
}

impl Matcher {
    /// Last matching pattern wins, like go-git's matcher.
    pub fn is_ignored(&self, path: &[String], is_dir: bool) -> bool {
        for pattern in self.patterns.iter().rev() {
            match pattern.matches(path, is_dir) {
                MatchResult::NoMatch => {}
                m => return m == MatchResult::Exclude,
            }
        }
        false
    }
}

fn read_patterns_file(path: &Path, domain: &[String], out: &mut Vec<Pattern>) {
    let Ok(content) = fs::read(path) else {
        return;
    };
    let text = String::from_utf8_lossy(&content);
    for line in text.split('\n') {
        let line = line.strip_suffix('\r').unwrap_or(line);
        if !line.starts_with('#') && !line.trim().is_empty() {
            out.push(Pattern::parse(line, domain));
        }
    }
}

/// Root-domain patterns only: `.git/info/exclude` first (lower priority), then
/// `.gitignore` — identical ordering to go-git `ReadPatterns` at the root.
pub fn load_root_matcher(cwd: &Path) -> Option<Matcher> {
    let mut patterns = Vec::new();
    read_patterns_file(&cwd.join(".git/info/exclude"), &[], &mut patterns);
    read_patterns_file(&cwd.join(".gitignore"), &[], &mut patterns);
    if patterns.is_empty() {
        return None;
    }
    Some(Matcher { patterns })
}

#[allow(dead_code)]
fn _unused(_: io::Error) {}
