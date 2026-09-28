//! Safe, confined signal-file reads. Ports `internal/signalfile` semantics:
//! project-relative paths only, no symlink traversal for content reads,
//! directories rejected, and a hard 1 MiB bound with a one-byte sentinel.

use std::fmt;
use std::fs;
use std::io;
use std::path::{Component, Path, PathBuf};

pub const MAX_READ_BYTES: usize = 1 << 20;
const MAX_SYMLINKS: usize = 255;

#[derive(Debug)]
pub enum ReadError {
    UnsafePath(String),
    Symlink(String),
    Directory(String),
    TooLarge(String),
    Io(io::Error),
}

impl fmt::Display for ReadError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::UnsafePath(name) => write!(f, "unsafe signal file path: {}", name),
            Self::Symlink(name) => write!(f, "signal file path contains symlink: {}", name),
            Self::Directory(name) => write!(f, "signal file path is a directory: {}", name),
            Self::TooLarge(name) => {
                write!(f, "signal file exceeds safe read limit: {}", name)
            }
            Self::Io(err) => write!(f, "{}", err),
        }
    }
}

impl std::error::Error for ReadError {}

impl From<io::Error> for ReadError {
    fn from(err: io::Error) -> Self {
        Self::Io(err)
    }
}

pub fn is_safe_skip(err: &ReadError) -> bool {
    matches!(
        err,
        ReadError::UnsafePath(_) | ReadError::Symlink(_) | ReadError::TooLarge(_)
    )
}

pub fn is_permission(err: &ReadError) -> bool {
    matches!(err, ReadError::Io(e) if e.kind() == io::ErrorKind::PermissionDenied)
}

pub fn is_not_found(err: &ReadError) -> bool {
    matches!(err, ReadError::Io(e) if e.kind() == io::ErrorKind::NotFound)
}

pub fn clean_relative_path(name: &str) -> Result<String, ReadError> {
    let normalized = name.trim().replace('\\', "/");
    if normalized.is_empty() {
        return Err(ReadError::UnsafePath(String::new()));
    }
    if normalized.starts_with('/') || normalized.starts_with("//") {
        return Err(ReadError::UnsafePath(name.to_string()));
    }
    if normalized.len() >= 2 && normalized.as_bytes()[1] == b':' {
        return Err(ReadError::UnsafePath(name.to_string()));
    }

    let cleaned = clean_path(&normalized);
    if !is_valid_path(&cleaned) {
        return Err(ReadError::UnsafePath(name.to_string()));
    }
    Ok(cleaned)
}

fn clean_path(p: &str) -> String {
    let mut out: Vec<&str> = Vec::new();
    for seg in p.split('/') {
        match seg {
            "" | "." => {}
            ".." => {
                if out.last().is_some_and(|last| *last != "..") {
                    out.pop();
                } else {
                    out.push("..");
                }
            }
            _ => out.push(seg),
        }
    }
    if out.is_empty() {
        ".".to_string()
    } else {
        out.join("/")
    }
}

fn is_valid_path(p: &str) -> bool {
    if p == "." {
        return true;
    }
    if p.is_empty() || p.starts_with('/') || p.ends_with('/') {
        return false;
    }
    p.split('/')
        .all(|seg| seg != "." && seg != ".." && !seg.is_empty())
}

fn reject_symlinks(root: &Path, cleaned: &str) -> Result<(), ReadError> {
    let mut current = PathBuf::from(root);
    for part in cleaned.split('/') {
        current.push(part);
        let info = fs::symlink_metadata(&current).map_err(ReadError::Io)?;
        if info.file_type().is_symlink() {
            return Err(ReadError::Symlink(cleaned.to_string()));
        }
    }
    Ok(())
}

/// Read `name` under `root`, never following symlinks and never reading more
/// than MAX_READ_BYTES + 1 bytes.
pub fn read_signal_file(root: &Path, name: &str) -> Result<Vec<u8>, ReadError> {
    let cleaned = clean_relative_path(name)?;
    reject_symlinks(root, &cleaned)?;

    let path = root.join(&cleaned);
    let file = fs::File::open(&path).map_err(ReadError::Io)?;
    let info = file.metadata().map_err(ReadError::Io)?;
    if info.is_dir() {
        return Err(ReadError::Directory(cleaned));
    }

    let mut limited = file.take(MAX_READ_BYTES as u64 + 1);
    let mut content = Vec::new();
    use std::io::Read;
    limited.read_to_end(&mut content).map_err(ReadError::Io)?;
    if content.len() > MAX_READ_BYTES {
        return Err(ReadError::TooLarge(cleaned));
    }
    Ok(content)
}

/// Resolve `rel` inside `root` the way Go's `os.Root` does: symlinks may be
/// followed, but every hop must stay lexically inside the root (absolute
/// symlink targets are treated as root-relative).
pub fn resolve_in_root(root: &Path, rel: &str) -> io::Result<PathBuf> {
    let mut resolved = PathBuf::from(root);
    let mut symlinks = 0usize;

    let mut pending: Vec<String> = rel.split('/').map(|s| s.to_string()).collect();
    while let Some(component) = pending.first().cloned() {
        pending.remove(0);
        match component.as_str() {
            "" | "." => continue,
            ".." => {
                if resolved == root {
                    return Err(io::Error::new(
                        io::ErrorKind::InvalidInput,
                        "path escapes root",
                    ));
                }
                resolved.pop();
                continue;
            }
            _ => {}
        }
        resolved.push(&component);
        let meta = fs::symlink_metadata(&resolved)?;
        if meta.file_type().is_symlink() {
            symlinks += 1;
            if symlinks > MAX_SYMLINKS {
                return Err(io::Error::other("too many links"));
            }
            let target = fs::read_link(&resolved)?;
            resolved.pop();
            let segs: Vec<String> = if target.is_absolute() {
                target
                    .components()
                    .filter_map(|c| match c {
                        Component::Normal(s) => Some(s.to_string_lossy().to_string()),
                        Component::ParentDir => Some("..".to_string()),
                        _ => None,
                    })
                    .collect()
            } else {
                target
                    .to_string_lossy()
                    .split('/')
                    .map(|s| s.to_string())
                    .collect()
            };
            let mut rest = segs;
            rest.extend(pending);
            pending = rest;
        }
    }
    if !resolved.starts_with(root) {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "path escapes root",
        ));
    }
    Ok(resolved)
}

fn has_meta(pattern: &str) -> bool {
    pattern.bytes().any(|b| matches!(b, b'*' | b'?' | b'['))
}

/// Port of Go `fs.Glob` over a confined directory root. Returns the matched
/// slash-separated relative paths in ReadDir (lexical) order.
pub fn glob_in_root(root: &Path, pattern: &str) -> Result<Vec<String>, io::Error> {
    // Pattern well-formedness check, like path.Match(pattern, "").
    if crate::globmatch::path_match(pattern, "").is_err() {
        return Err(io::Error::other("syntax error in pattern"));
    }
    if !has_meta(pattern) {
        return match stat_in_root(root, pattern) {
            Ok(_) => Ok(vec![pattern.to_string()]),
            Err(_) => Ok(Vec::new()),
        };
    }

    let (dir, file) = match pattern.rfind('/') {
        Some(idx) => (&pattern[..idx], &pattern[idx + 1..]),
        None => (".", pattern),
    };
    let dir = if dir.is_empty() { "." } else { dir };

    if !has_meta(dir) {
        return glob_dir(root, dir, file, Vec::new());
    }

    let dirs = glob_in_root(root, dir)?;
    let mut matches = Vec::new();
    for d in dirs {
        matches = glob_dir(root, &d, file, matches)?;
    }
    Ok(matches)
}

fn glob_dir(
    root: &Path,
    dir: &str,
    file_pattern: &str,
    mut matches: Vec<String>,
) -> Result<Vec<String>, io::Error> {
    let resolved = match resolve_in_root(root, dir) {
        Ok(p) => p,
        Err(_) => return Ok(matches),
    };
    let entries = match fs::read_dir(&resolved) {
        Ok(e) => e,
        Err(_) => return Ok(matches), // ignore I/O error, like Go
    };
    let mut names: Vec<String> = Vec::new();
    for entry in entries {
        let entry = entry?;
        names.push(entry.file_name().to_string_lossy().to_string());
    }
    names.sort();
    for name in names {
        match crate::globmatch::path_match(file_pattern, &name) {
            Ok(true) => matches.push(if dir == "." {
                name
            } else {
                format!("{}/{}", dir, name)
            }),
            Ok(false) => {}
            Err(_) => return Err(io::Error::other("syntax error in pattern")),
        }
    }
    Ok(matches)
}

fn stat_in_root(root: &Path, rel: &str) -> io::Result<fs::Metadata> {
    let resolved = resolve_in_root(root, rel)?;
    fs::metadata(&resolved)
}
