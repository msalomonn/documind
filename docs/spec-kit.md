# Spec Kit integration

Installed: `specify-cli==1.2.3` from its official PyPI distribution, Python 3.12.14, on 2026-10-25. The local CLI lives in an isolated workspace virtual environment; it does not change the system Python. Managed assets are committed, so readers can inspect them without a global installation.

Setup used:

```sh
specify init --here --force --non-interactive --integration codex \
  --integration-options="--skills --events false" --script sh --ignore-agent-tools
```

This was run only against the new foundation directories. The existing constitution was preserved. For a fresh clone, do not blindly repeat `init --force`: inspect the integration manifest and use the documented upgrade path when needed.

## CLI setup on another machine

With Python 3.11 or newer, create an isolated environment outside the repository, install `specify-cli==1.2.3`, and run `specify version`. See the [official installation guide](https://github.github.com/spec-kit/installation.html). The exact dependency snapshot used for bootstrap is in portfolio-engineering's `tooling/spec-kit-requirements.lock`; it records this environment, not application dependencies.

## Commands and state

The Codex integration installed ten skills in `.agents/skills/`; `specify integration status` checks their manifest state. Runtime event handlers were disabled. No Caronte hooks, git/Linear bridge, autonomous workflow or automatic merge was installed.

In a project session, use `$speckit-specify`, `$speckit-plan`, `$speckit-tasks`, `$speckit-analyze` and `$speckit-converge` as needed. Initial constitution/spec approval remains pending. Installation checks do not prove the desktop client has discovered these skills or that a feature has been implemented.

Spec Kit selects the active feature using `.specify/feature.json`, which upstream intentionally ignores in Git. On a fresh clone, set the feature explicitly before running scripts:

```sh
export SPECIFY_FEATURE_DIRECTORY=specs/001-document-pipeline
bash .specify/scripts/bash/check-prerequisites.sh --json --require-spec --require-tasks --include-tasks
```

The bootstrap checkout also has the local pointer set to this feature. Changing Git branches alone does not select a different Spec Kit feature.

References: [SDD quickstart](https://github.github.com/spec-kit/quickstart.html), [Codex integration](https://github.github.com/spec-kit/integrations/codex.html). Commands verified against the installed CLI help and bundled scripts.
