# Go Qase Testing Reporter

This is a command line to read testing report file in JSON Lines format and send reporting to Qase. Since Go built-in testing library does not provide any listener, this tool will be run after the test is finished and generated the JSON Lines report file.

## 1. Installation

To install the command run the following command:

```bash
go install github.com/petrabarus/go-qase-testing-reporter
```

## 2. Usage

### 2.1. Configuration

Before running the command you need to pass the configuration in the environment variable. We use the same environment variable as the Qase official libraries. Below is the list of the environment variables that you need to pass:

- `QASE_TESTOPS_PROJECT` The project code in Qase.
- `QASE_TESTOPS_API_TOKEN` The API token in Qase.
- `QASE_TESTOPS_RUN_TITLE` The name of the run in Qase.

### 2.2. Run the command

Once you have the configuration, you can run the command by passing the path to the JSON report file. You can also use flags as alternatives of environment variables. Below is an example of the command:

```bash
go-qase-testing-reporter \
    --api-token <Qase API token> \ 
    --project <Qase project code> \
    --run-title <Qase run ID> \
    <path/to/report.jsonl>
```

You can also use the Qase's official environment variables.

```
QASE_TESTOPS_API_TOKEN=XXX \
    QASE_TESTOPS_PROJECT=XXX \
    QASE_TESTOPS_RUN_TITLE="Test Run $(date +%Y-%m-%d %H:%M:%S)" \
    go-qase-testing-reporter report.jsonl
```

The command above will read JSON Lines file `path/to/report.jsonl` and send the report to Qase.

### 2.3. Output

The command will output the following information:

1. The ID of the run in Qase.

The format will be in JSON like below:

```json
{
    "run_id": "1234567890"
}
```