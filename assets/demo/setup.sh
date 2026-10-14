#!/usr/bin/env bash
# Builds a throwaway workspace of git repos for assets/demo.tape.
# Usage: assets/demo/setup.sh <dir>
set -euo pipefail

D=$1
rm -rf "$D"
mkdir -p "$D/code" "$D/remotes"

export GIT_CONFIG_NOSYSTEM=1
export GIT_CONFIG_GLOBAL="$D/.gitconfig"
git config --global user.name "Demo User"
git config --global user.email "demo@example.com"
git config --global init.defaultBranch main
git config --global advice.detachedHead false

commit() { # commit <days-ago> <message>
  local when
  when=$(date -v-"$1"d +%s 2>/dev/null || date -d "-$1 days" +%s)
  GIT_AUTHOR_DATE="@$when" GIT_COMMITTER_DATE="@$when" git commit -qm "$2"
}

new_repo() { # new_repo <name>
  git init -q --bare "$D/remotes/$1.git"
  git clone -q "$D/remotes/$1.git" "$D/code/$1" 2>/dev/null
  cd "$D/code/$1"
  printf '# %s\n' "$1" >README.md
  git add -A && commit 6 "initial commit"
  for i in $(seq 1 "$2"); do
    echo "line $i" >>notes.txt
    git add -A && commit $((i % 6)) "update notes $i"
  done
  git push -q origin main
}

new_repo api 9
new_repo web 5
new_repo cli 3
new_repo docs 2

# api: work in progress, plus a linked worktree for a feature branch
cd "$D/code/api"
seq 1 40 >handlers.go
echo "changed" >>README.md
git add handlers.go
echo "scratch" >todo.txt
git worktree add -q -b feature/auth "$D/code/api-auth"
cd "$D/code/api-auth"
seq 1 25 >auth.go
git add auth.go && commit 0 "add auth middleware"

# web: two local commits not yet pushed
cd "$D/code/web"
seq 1 60 >app.ts && git add -A && commit 1 "add app shell"
seq 1 30 >routes.ts && git add -A && commit 0 "add routes"

# cli: behind the remote
cd "$D/code/cli"
git clone -q "$D/remotes/cli.git" "$D/tmp-cli" 2>/dev/null
cd "$D/tmp-cli"
echo "fix" >>notes.txt && git add -A && commit 0 "fix flag parsing" && git push -q origin main
rm -rf "$D/tmp-cli"
cd "$D/code/cli" && git fetch -q

# docs: merge conflict
cd "$D/code/docs"
git switch -q -c rewrite
echo "rewrite" >README.md && git add -A && commit 0 "rewrite intro"
git switch -q main
echo "typo fix" >README.md && git add -A && commit 0 "fix typo"
git merge -q rewrite >/dev/null 2>&1 || true

# Point origins at a public-looking URL for display; nothing fetches during the demo.
for r in api web cli docs; do
  git -C "$D/code/$r" remote set-url origin "https://github.com/acme/$r.git"
done

mkdir -p "$D/.config/gv"
cat >"$D/.config/gv/config.yaml" <<YAML
scan_paths:
  - ~/code
poll_interval: 1s
YAML
