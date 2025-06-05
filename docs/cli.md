- [Build](#build)
- [Configuration](#configuration)
- [Create and update resources](#create-and-update-resources)
- [Describe resources](#describe-resources)
- [List events](#list-events)
- [Approve events](#approve-events)

The CLI accepts YAMLs to define the resources for your superplane. The examples in the [docs/examples](./docs/examples) folder should have you covered on what those YAMLs look like.

### Build

```bash
# Building for Mac ARMs
make cli.build OS=darwin ARCH=arm64

# Building for Linux x86_64
make cli.build OS=linux ARCH=amd64
```

This will build the CLI binary in `build/cli`.

### Configuration

By default, the CLI points to the local SuperPlane application, running at `http://localhost:8080`. You can update that configuration with:

```bash
/build/cli config set api_url <API_URL>
```

### Create and update resources

To create resources, you use the `create` command:

```bash
./build/cli create -f ./docs/examples/stage.yaml
```

To update resources, you use the `update` command:

```bash
./build/cli update -f ./docs/examples/stage.yaml
```

### Describe resources

To describe resources, you use the `get` command:

```bash
./build/cli get canvas <canvas_id_or_name>
./build/cli get stage <stage_id_or_name> --canvas-name <canvas_name>
./build/cli get event-source <event_source_name_or_id> --canvas-name <canvas_name>
./build/cli get secret <secret_id_or_name> --canvas-name <canvas_name>
```

### List events

To list events for a stage, you use the `list` command:

```bash
./build/cli list events --stage-name <stage_name> --canvas-name <canvas_name>
```

### Approve events

To approve events for a stage, you use the `approve` command:

```bash
./build/cli approve event <event_id> --stage-name <stage_name> --canvas-name <canvas_name>
```