# cocli example collections

This is a collection of real-world cocli use cases.

## Examples

1. `watch-and-upload`: Watch a given folder and anytime a given file is moved or created in the folder, it will automatically upload the files to a given record split by hours.

### Ray Action

Set the worker image and Python targets in [ray-action.yaml](ray-action.yaml).
Run `cocli action create -f examples/ray-action.yaml --dry-run` to preview the
spec, then `cocli action create -p <project> -f examples/ray-action.yaml`.
Use `cocli action update <action-id> -p <project> -f examples/ray-action.yaml`
to replace the definition and `cocli action run <action-id> <record-id> -p <project>`
to execute it.
