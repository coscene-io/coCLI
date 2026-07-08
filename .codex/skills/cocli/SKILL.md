---
name: cocli
description: Use when an AI agent needs to operate or explain the CoScene CLI (cocli), including login profiles and access tokens, API endpoints, projects, records, record files, project files, labels, custom fields, actions, action runs, uploads, downloads, and headless scripting patterns.
---

# CoScene CLI

Use `cocli` as the first tool for CoScene project, record, file, and action work. Prefer live help over assumptions because flags and output fields can change:

```bash
cocli --version
cocli <command> --help
cocli <command> <subcommand> --help
```

For automation, prefer `--no-tty` when available and `-o json` output so scripts can parse stable data with `jq`.

## Core Concepts

- **Endpoint**: the OpenAPI host for a CoScene environment. Common values are `https://openapi.coscene.io` and `https://openapi.coscene.cn`. Confirm the endpoint with `cocli --version`, `cocli login current`, and `cocli login list -v`.
- **Profile**: a local named login entry containing endpoint, token, organization, and default project. Profiles live in the local cocli config, not in a repository.
- **Token**: an access token used by cocli. Never print it, commit it, or put it in shared logs.
- **Organization**: the workspace namespace that owns projects.
- **Project**: the main data boundary. Commands usually accept a project slug with `-p <project-slug>`. Some JSON responses use resource names like `projects/<uuid>`.
- **Record**: a dataset container inside a project. Record commands accept either a record id or a resource name like `projects/<uuid>/records/<uuid>`.
- **Record files**: files stored under a record. Upload paths decide where files appear in the record.
- **Project files**: files stored at project scope rather than under a specific record.
- **Labels**: lightweight searchable metadata. Prefer normalized `key: value` strings when preserving source semantics.
- **Custom fields**: typed project schema fields. Use them only after checking the project schema.
- **Actions**: reusable compute definitions available to a project or as system templates.
- **Action runs**: executions of actions against records.

## Authentication

Create or update a profile:

```bash
cocli login set \
  -n <profile-name> \
  -e <openapi-endpoint> \
  -t "$COS_TOKEN" \
  -p <default-project-slug>
```

Useful checks:

```bash
cocli login current
cocli login list -v
cocli project list --all -o json
```

For ephemeral CI-style runs, cocli can load credentials from environment variables:

```bash
export COS_ENDPOINT=https://openapi.coscene.io
export COS_TOKEN=<access-token>
export COS_PROJECT=<project-slug>
cocli project list -o json
```

Keep tokens in environment variables, secret stores, or stdin-driven scripts. Avoid inline literal tokens in commands that may enter shell history.

## Projects

List and inspect projects before operating on records:

```bash
cocli project list --all -o json
```

When a command supports `-p`, pass the project slug explicitly in scripts instead of relying on the current profile default.

## Records

Create a record with stable metadata:

```bash
cocli record create \
  --no-tty \
  -p <project-slug> \
  -t "<record title>" \
  -d "<record description>" \
  -l "source: gcs" \
  -l "scenario: demo" \
  -o json
```

List, search, and inspect records:

```bash
cocli record list -p <project-slug> --all -o json
cocli record list -p <project-slug> --keywords "<keyword>" -o json
cocli record describe -p <project-slug> <record-id-or-name> -o json
```

Update metadata only after checking the current record:

```bash
cocli record update -p <project-slug> <record-id-or-name> --help
```

## Record Files

Upload selected files explicitly when the record root layout matters:

```bash
cocli record upload \
  --no-tty \
  -p <project-slug> \
  <record-id-or-name> \
  ./metadata.json ./tracking.jsonl ./video.mp4
```

To upload a directory, pass the directory path as an upload source. Use `--dir` only to choose the remote target directory:

```bash
cocli record upload --no-tty -p <project-slug> <record-id-or-name> ./recording_dir
cocli record upload --no-tty -p <project-slug> <record-id-or-name> --dir backup/ ./metadata.json
```

Verify and download record files:

```bash
cocli record file list -p <project-slug> <record-id-or-name> -R --all -o json
cocli record download -p <project-slug> <record-id-or-name> ./downloads
```

For large uploads, check `cocli record upload --help` for concurrency and part-size flags such as `-P` and `-s`.

## Project Files

Use project file commands for shared files that do not belong to one record:

```bash
cocli project file list <project-slug> -R --all -o json
cocli project file upload --no-tty <project-slug> <local-path>
cocli project file download <project-slug> <local-destination> --dir <remote-path>
```

Confirm exact flags with `cocli project file <subcommand> --help`.

## Actions

List available actions:

```bash
cocli action list -p <project-slug> -o json
```

Run an action against a record:

```bash
cocli action run \
  -p <project-slug> \
  <action-id-or-name> \
  <record-id-or-name> \
  --skip-params \
  -f
```

Pass action parameters with repeated `-P key=value` flags when needed. Track runs with:

```bash
cocli action list-run -p <project-slug> -r <record-id-or-name> -o json
```

The CLI supports listing actions and creating action runs. If an automation needs to create or edit action definitions, inspect the current API or use the CoScene web console; do not assume `cocli action` has a create command.

## Scripting Checklist

1. Run `cocli --version` and confirm the release channel and base API endpoint.
2. Run `cocli login current` and `cocli login list -v` to confirm endpoint, organization, and default project.
3. Pass `-p <project-slug>` explicitly in scripts.
4. Use `--no-tty` for non-interactive create/upload commands.
5. Use `-o json` and parse ids/resource names from JSON output.
6. Keep tokens and temporary manifests out of commits and uploaded record files.
7. Verify uploads with `cocli record file list -R --all -o json`.
8. For repeatable migrations, store source identifiers in labels, descriptions, or custom fields before path context is lost.
