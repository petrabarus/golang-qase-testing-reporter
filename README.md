# golang-qase-testing-reporter

This is a command line to read testing report file in JSON Lines format and send reporting to Qase. Since Go built-in testing library does not provide any listener, this tool will be run after the test is finished and generated the JSON Lines report file.

## Installation

To install the command run the following command:

```bash
go install github.com/petrabarus/golang-qase-testing-reporter
```

## Usage

### Configuration

Before running the command you need to pass the configuration in the environment variable. We use the same environment variable as the Qase official libraries. Below is the list of the environment variables that you need to pass:

- `QASE_PROJECT_CODE` The project code in Qase.
- `QASE_API_TOKEN` The API token in Qase.
- `QASE_API_BASE_URL` The base URL of the Qase API. Default is `https://api.qase.io/v1`.
- `QASE_RUN_NAME` The name of the run in Qase.


### Run the command

Once you have the configuration, you can run the command by passing the path to the JSON report file. Below is an example of the command:

```bash
golang-qase-testing-reporter path/to/report.jsonl
```

The command above will read JSON Lines file `path/to/report.jsonl` and send the report to Qase.