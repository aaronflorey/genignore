//! Byte-faithful port of Go's `path.Match` shell pattern matching, which is
//! also what `filepath.Match`/`filepath.Glob`/`fs.Glob` use on Linux.

use std::fmt;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BadPattern;

impl fmt::Display for BadPattern {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("syntax error in pattern")
    }
}

impl std::error::Error for BadPattern {}

type MatchResult = Result<bool, BadPattern>;

pub fn path_match(pattern: &str, name: &str) -> MatchResult {
    let mut pattern = pattern.as_bytes();
    let mut name = name.as_bytes();

    'outer: while !pattern.is_empty() {
        let (star, chunk, rest) = scan_chunk(pattern);
        pattern = rest;
        if star && chunk.is_empty() {
            return Ok(!name.contains(&b'/'));
        }

        match match_chunk(chunk, name) {
            Err(e) => {
                if e != BAD {
                    return Err(BadPattern);
                }
            }
            Ok((t, ok)) => {
                if ok && (t.is_empty() || !pattern.is_empty()) {
                    name = t;
                    continue;
                }
            }
        }

        if star {
            let mut i = 0usize;
            while i < name.len() && name[i] != b'/' {
                match match_chunk(chunk, &name[i + 1..]) {
                    Ok((t, ok)) => {
                        if ok {
                            if pattern.is_empty() && !t.is_empty() {
                                i += 1;
                                continue;
                            }
                            name = t;
                            continue 'outer;
                        }
                    }
                    Err(_) => return Err(BadPattern),
                }
                i += 1;
            }
        }

        while !pattern.is_empty() {
            let (_, chunk2, rest2) = scan_chunk(pattern);
            pattern = rest2;
            if match_chunk(chunk2, b"").is_err() {
                return Err(BadPattern);
            }
        }
        return Ok(false);
    }
    Ok(name.is_empty())
}

fn scan_chunk(pattern: &[u8]) -> (bool, &[u8], &[u8]) {
    let mut pattern = pattern;
    let mut star = false;
    while !pattern.is_empty() && pattern[0] == b'*' {
        pattern = &pattern[1..];
        star = true;
    }
    let mut inrange = false;
    let mut i = 0usize;
    while i < pattern.len() {
        match pattern[i] {
            b'\\' => {
                if i + 1 < pattern.len() {
                    i += 1;
                }
            }
            b'[' => inrange = true,
            b']' => inrange = false,
            b'*' if !inrange => return (star, &pattern[..i], &pattern[i..]),
            _ => {}
        }
        i += 1;
    }
    (star, pattern, b"")
}

const BAD: i32 = 1;

fn match_chunk<'a>(mut chunk: &[u8], mut s: &'a [u8]) -> Result<(&'a [u8], bool), i32> {
    let mut failed = false;
    while !chunk.is_empty() {
        failed = failed || s.is_empty();
        match chunk[0] {
            b'[' => {
                let mut r: u32 = 0;
                if !failed {
                    let (c, n) = decode_rune(s);
                    r = c;
                    s = &s[n..];
                }
                chunk = &chunk[1..];
                let mut negated = false;
                if !chunk.is_empty() && chunk[0] == b'^' {
                    negated = true;
                    chunk = &chunk[1..];
                }
                let mut matched = false;
                let mut nrange = 0usize;
                loop {
                    if !chunk.is_empty() && chunk[0] == b']' && nrange > 0 {
                        chunk = &chunk[1..];
                        break;
                    }
                    let (lo, rest) = get_esc(chunk).map_err(|_| BAD)?;
                    chunk = rest;
                    let mut hi = lo;
                    if chunk.is_empty() {
                        return Err(BAD);
                    }
                    if chunk[0] == b'-' {
                        let (h, rest2) = get_esc(&chunk[1..]).map_err(|_| BAD)?;
                        hi = h;
                        chunk = rest2;
                    }
                    matched = matched || (lo <= r && r <= hi);
                    nrange += 1;
                }
                failed = failed || matched == negated;
            }
            b'?' => {
                if !failed {
                    failed = s[0] == b'/';
                    let (_, n) = decode_rune(s);
                    s = &s[n..];
                }
                chunk = &chunk[1..];
            }
            b'\\' => {
                chunk = &chunk[1..];
                if chunk.is_empty() {
                    return Err(BAD);
                }
                if !failed {
                    failed = chunk[0] != s[0];
                    s = &s[1..];
                }
                chunk = &chunk[1..];
            }
            _ => {
                if !failed {
                    failed = chunk[0] != s[0];
                    s = &s[1..];
                }
                chunk = &chunk[1..];
            }
        }
    }
    if failed {
        return Ok((b"", false));
    }
    Ok((s, true))
}

fn get_esc(chunk: &[u8]) -> Result<(u32, &[u8]), ()> {
    if chunk.is_empty() || chunk[0] == b'-' || chunk[0] == b']' {
        return Err(());
    }
    let chunk = if chunk[0] == b'\\' {
        let rest = &chunk[1..];
        if rest.is_empty() {
            return Err(());
        }
        rest
    } else {
        chunk
    };
    let (r, n) = decode_rune(chunk);
    if r == 0xFFFD && n == 1 && chunk[0] >= 0x80 {
        return Err(());
    }
    let rest = &chunk[n..];
    if rest.is_empty() {
        return Err(());
    }
    Ok((r, rest))
}

fn decode_rune(s: &[u8]) -> (u32, usize) {
    if s.is_empty() {
        return (0xFFFD, 0);
    }
    let len = utf8_len(s[0]);
    if len == 1 {
        return (s[0] as u32, 1);
    }
    if s.len() < len {
        return (0xFFFD, 1);
    }
    match std::str::from_utf8(&s[..len]) {
        Ok(st) => {
            let c = st.chars().next().unwrap_or('\u{FFFD}');
            (c as u32, len)
        }
        Err(_) => (0xFFFD, 1),
    }
}

fn utf8_len(b: u8) -> usize {
    if b < 0xC0 {
        // ASCII and continuation bytes both consume one input byte on decode.
        1
    } else if b < 0xE0 {
        2
    } else if b < 0xF0 {
        3
    } else {
        4
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn star_matches_within_segment() {
        assert!(path_match("*.tf", "main.tf").unwrap());
        assert!(!path_match("*.tf", "main.tfvars").unwrap());
        assert!(!path_match("*", "x/y").unwrap());
        assert!(path_match("*", "name").unwrap());
    }

    #[test]
    fn classes_and_escapes() {
        assert!(path_match("[a-c]x", "bx").unwrap());
        assert!(!path_match("[^a-c]x", "bx").unwrap());
        assert!(path_match("[^a-c]x", "dx").unwrap());
        assert!(path_match("\\*x", "*x").unwrap());
    }

    #[test]
    fn question_mark() {
        assert!(path_match("?.iml", "a.iml").unwrap());
        assert!(!path_match("?", "ab").unwrap());
    }

    #[test]
    fn multi_star() {
        assert!(path_match("*.tf*", "x.tfx").unwrap());
        assert!(path_match("*a*b", "aab").unwrap());
    }
}
