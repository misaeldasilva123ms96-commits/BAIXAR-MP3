# Agent instructions

## Git worktree policy

All Git worktrees for this repository MUST be created under:

```text
D:\Dev\Worktrees
```

Required directory naming:

```text
D:\Dev\Worktrees\BAIXAR-MP3-<sanitized-branch-name>
```

Sanitize only the directory name. The actual Git branch name MUST NOT be
changed. Replace `/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`,
and whitespace with `-`. Collapse repeated `-` characters and trim leading
or trailing `-` characters from the sanitized component.

Always verify that the destination does not already exist and pass its absolute
path explicitly to `git worktree add`.

For a new branch:

```powershell
git worktree add "D:\Dev\Worktrees\BAIXAR-MP3-feat-example" -b "feat/example"
```

For an existing branch:

```powershell
git worktree add "D:\Dev\Worktrees\BAIXAR-MP3-feat-example" "feat/example"
```

Never create worktrees:

- inside the repository;
- under `.worktrees`;
- under a relative `worktrees` directory;
- beside the repository implicitly;
- under `C:`;
- under `%TEMP%`;
- outside `D:\Dev\Worktrees`;
- with an implicit or relative target path.

Never move a registered worktree manually with Windows Explorer. Use
`git worktree move`.
