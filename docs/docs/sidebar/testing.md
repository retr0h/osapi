---
sidebar_position: 3
---

# Testing

Enable [Remote Taskfile][] feature.

```bash
export TASK_X_REMOTE_TASKFILES=1
```

Install dependencies:

```bash
$ task deps
```

To execute tests:

```bash
$ task test
```

Auto format code:

```bash
$ task fmt
```

List helpful targets:

```bash
$ task
```

[Remote Taskfile]: https://taskfile.dev/experiments/remote-taskfiles/
