// go-qase-testing-reporter is a tool to report test results to Qase.
// Since Go testing does not have a built-in testing event listener,
// we need to parse the test output and report the results to Qase.
// To run this file locally, execute:
// go run . --api-token <qase-api-token> --project <qase-project-name> --run-title <qase-run-title> <path/to/test-results.jsonl>
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	qase "go.qase.io/client"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type Config struct {
	Filename     string
	QaseApiToken string `mapstructure:"api_token"`
	QaseProject  string `mapstructure:"project"`
	QaseRunTitle string `mapstructure:"run_title"`
}

type ReportJsonLine struct {
	Time    string  `json:"time"`
	Test    string  `json:"test"`   // The name of the test
	Action  string  `json:"action"` // The action of the test, we shall check for "pass" or "fail"
	Package string  `json:"package"`
	Output  string  `json:"output"`
	Elapsed float64 `json:"elapsed"`
}

type ReportResult struct {
	Package    string
	TestCaseId int64
	Status     string
	Time       time.Time
	TimeMs     int64
}

type ReportResultOutput struct {
	TestCaseId int64
	Status     string
}

type ReportOutput struct {
	RunId    int32                 `json:"run_id"`
	TestRuns []ReportOutputTestRun `json:"test_runs"`
}

type ReportOutputTestRun struct {
	TestCaseId int64  `json:"test_case_id"`
	Status     string `json:"status"`
}

var (
	ctx context.Context

	config Config

	cmd = &cobra.Command{
		Use:   "go-qase-testing-reporter <filename>",
		Short: "go-qase-testing-reporter is a tool to report test results to Qase",
		Long: `go-qase-testing-reporter is a tool to report test results to Qase.
Since Go testing does not have a built-in testing event listener, 
we need to parse the test output and report the results to Qase.
`,
		Args:             cobra.MaximumNArgs(1),
		ArgAliases:       []string{"filename"},
		PersistentPreRun: preRun,
		Run:              RunCommand,
	}

	qaseClient qase.APIClient
)

const (
	TEST_CASE_RESULT_STATUS_PASSED = "passed"
	TEST_CASE_RESULT_STATUS_FAILED = "failed"
)

func init() {
	cobra.OnInitialize()

	cmd.Flags().StringP("project", "p", "", "Qase project name")
	cmd.Flags().StringP("api-token", "t", "", "Qase API token")
	cmd.Flags().StringP("run-title", "r", "", "Qase run title")

	// add --version flag
	cmd.Flags().BoolP("version", "v", false, "Print version")

	viper.BindPFlag("project", cmd.Flags().Lookup("project"))
	viper.BindPFlag("api_token", cmd.Flags().Lookup("api-token"))
	viper.BindPFlag("run_title", cmd.Flags().Lookup("run-title"))

	// Adopts the official Qase environment variables
	viper.BindEnv("project", "QASE_TESTOPS_PROJECT")
	viper.BindEnv("api_token", "QASE_TESTOPS_API_TOKEN")
	viper.BindEnv("run_title", "QASE_TESTOPS_RUN_TITLE")
}

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func preRun(cmd *cobra.Command, args []string) {
	viper.AutomaticEnv()
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to read Viper options into configuration: %v", err)
	}
	if len(args) > 0 {
		config.Filename = args[0]
	}

	//log.Printf("Config: %+v", config)
	ctx = context.Background()

	initQaseClient()
}

func initQaseClient() {
	configuration := qase.NewConfiguration()
	configuration.AddDefaultHeader("Token", config.QaseApiToken)
	qaseClient = *qase.NewAPIClient(configuration)
}

func RunCommand(cmd *cobra.Command, args []string) {
	if printVersion(cmd) {
		return
	}

	if config.Filename == "" {
		fmt.Fprintln(os.Stderr, "Error: filename is required")
		// print usage
		cmd.Usage()
		return
	}

	var err error
	var output ReportOutput
	//fmt.Println("Running go-qase-testing-reporter")
	results, err := processFile(config.Filename)
	if err != nil {
		log.Fatalf("Failed to process file: %v", err)
	}

	//	log.Printf("Results: %+v", results)
	id, err := createNewRun(results)
	if err != nil {
		log.Fatalf("Failed to create test run: %v", err)
	}

	testRunResultOutputs, err := createTestRunResults(id, results)
	if err != nil {
		log.Fatalf("Failed to create test run result: %v", err)
	}

	err = completeRun(id)
	if err != nil {
		log.Fatalf("Failed to complete test run: %v", err)
	}

	output = createOutput(id, testRunResultOutputs)
	printOutput(output)
}

func printVersion(cmd *cobra.Command) (shouldExit bool) {
	shouldPrintVersion, _ := cmd.Flags().GetBool("version")
	if !shouldPrintVersion {
		return false
	}
	version, ok := getVersionFromBuildInfo()
	if ok {
		version = fmt.Sprintf("%s-%s-%s", Version, Commit, Date)
	}
	fmt.Printf("go-qase-testing-reporter %s\n", version)
	return true
}

// Enable to generate version if being installed using `go install`
func getVersionFromBuildInfo() (version string, ok bool) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	version = buildInfo.Main.Version
	// find `vcs.time` and `vcs.revision`
	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.time" {
			// truncate to 10 chars if possible
			date := setting.Value
			if len(date) > 10 {
				date = date[:10]
			}
			version = fmt.Sprintf("%s-%s", version, date)
		}
		if setting.Key == "vcs.revision" {
			// truncate to 8 chars if possible
			commit := setting.Value
			if len(commit) > 8 {
				commit = commit[:8]
			}
			version = fmt.Sprintf("%s-%s", version, commit)
		}
	}
	return
}

func createNewRun(results []ReportResult) (runId int32, err error) {
	// Create Test Run
	caseIds := make([]int64, 0)
	for _, result := range results {
		caseIds = append(caseIds, result.TestCaseId)
	}

	qaseResp, httpResp, err := qaseClient.RunsApi.CreateRun(ctx, qase.RunCreate{
		Title: config.QaseRunTitle,
		Cases: caseIds,
	}, config.QaseProject)
	if err != nil {
		err = fmt.Errorf("failed to create test run: %v", err)
		return
	}

	if httpResp.StatusCode != 200 {
		err = fmt.Errorf("failed to create test run, status code: %v", httpResp.StatusCode)
		return
	}

	runId = int32(qaseResp.Result.Id)
	return
}

func createTestRunResults(runId int32, results []ReportResult) (testRunResultOutputs []ReportResultOutput, err error) {
	testRunResultOutputs = make([]ReportResultOutput, 0)
	qaseResults := make([]qase.ResultCreate, 0)
	for _, result := range results {
		qaseResult := qase.ResultCreate{
			CaseId: int64(result.TestCaseId),
			Status: result.Status,
			// Somewhat this result in bad request
			//Time:   result.Time.Unix(),
			TimeMs: result.TimeMs,
		}
		if result.Package != "" {
			qaseResult.Comment = fmt.Sprintf("Package: %v", result.Package)
		}
		qaseResults = append(qaseResults, qaseResult)
		testRunResultOutputs = append(testRunResultOutputs, ReportResultOutput{
			TestCaseId: int64(result.TestCaseId),
			Status:     result.Status,
		})
	}

	qaseResp, httpResp, err := qaseClient.ResultsApi.CreateResultBulk(ctx, qase.ResultCreateBulk{
		Results: qaseResults,
	}, config.QaseProject, runId)

	if err != nil {
		// read body to string
		message, _ := io.ReadAll(httpResp.Body)
		err = fmt.Errorf("failed to create test run results: %v %s", err, message)
		return
	}

	if httpResp.StatusCode != 200 {
		message, _ := io.ReadAll(httpResp.Body)
		err = fmt.Errorf("failed to create test run results, status code: %v %s", httpResp.StatusCode, message)
		return
	}

	if !qaseResp.Status {
		err = fmt.Errorf("failed to create test run results, status false")
		return
	}

	return
}

func completeRun(id int32) (err error) {
	// Complete Test Run
	qaseResp, httpResp, err := qaseClient.RunsApi.CompleteRun(
		ctx,
		config.QaseProject,
		id,
	)
	if err != nil {
		err = fmt.Errorf("failed to complete test run: %v", err)
		return
	}

	if httpResp.StatusCode != 200 {
		err = fmt.Errorf("failed to complete test run, status code: %v", httpResp.StatusCode)
		return
	}

	if !qaseResp.Status {
		err = fmt.Errorf("failed to complete test run, status false")
		return
	}

	return nil
}

// There is a max of 2000 result per bulk request API.
// Once we reach the limit, we will update the code to send the results in multiple bulk requests.
func processFile(filename string) (results []ReportResult, err error) {
	file, err := os.Open(filename)
	if err != nil {
		err = errors.Join(errors.New("failed to open file"), err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	results = make([]ReportResult, 0)
	for scanner.Scan() {
		result, err := processLine(scanner.Text())
		if err != nil {
			//log.Printf("Failed to process line: %v", err)
			continue
		}
		if result.TestCaseId == 0 {
			continue
		}
		results = append(results, result)
		if len(results) == 2000 {
			return results, fmt.Errorf("max bulk request limit reached")
		}
	}

	if err = scanner.Err(); err != nil {
		err = errors.Join(errors.New("failed to read file"), err)
		return
	}

	return
}

func processLine(line string) (result ReportResult, err error) {
	var content ReportJsonLine
	err = json.Unmarshal([]byte(line), &content)
	if err != nil {
		err = errors.Join(errors.New("failed to parse line"), err)
		return
	}
	if content.Test == "" {
		err = fmt.Errorf("no test name found in line: %v", line)
		return
	}

	qaseId, err := ParseQaseId(content.Test)
	if err != nil {
		err = errors.Join(fmt.Errorf("failed to parse Qase ID in line: %v", line), err)
		return
	}
	if qaseId == 0 {
		err = fmt.Errorf("no Qase ID found in test name: %v", content.Test)
		return
	}
	result.TestCaseId = int64(qaseId)

	if content.Action == "fail" {
		result.Status = TEST_CASE_RESULT_STATUS_FAILED
		// test failed
	} else if content.Action == "pass" {
		result.Status = TEST_CASE_RESULT_STATUS_PASSED
		// test passed
	} else {
		err = fmt.Errorf("unknown action: %v", content.Action)
		return
	}

	if content.Time != "" {
		result.Time, err = time.Parse(time.RFC3339, content.Time)
		if err != nil {
			err = errors.Join(fmt.Errorf("failed to parse time: %v", content.Time), err)
			return
		}
		result.Time = result.Time.UTC()
	}

	if content.Elapsed != 0 {
		// convert to ms
		fmt.Printf("Elapsed: %v\n", content.Elapsed)
		result.TimeMs = int64(content.Elapsed * 1000)
		fmt.Printf("Elapsed: %v\n", result.TimeMs)
	}

	if content.Package != "" {
		result.Package = content.Package
	}

	return
}

func ParseQaseId(test string) (int, error) {
	re := regexp.MustCompile(`QASE-(\d+)`)
	matches := re.FindAllStringSubmatch(test, -1)
	if len(matches) == 0 {
		return 0, nil
	}
	lastMatch := matches[len(matches)-1]
	qaseId, err := strconv.Atoi(lastMatch[1])
	if err != nil {
		return 0, errors.New("failed to parse Qase ID")
	}
	return qaseId, nil
}

func createOutput(runId int32, testRunResultOutputs []ReportResultOutput) (output ReportOutput) {
	output = ReportOutput{
		RunId:    runId,
		TestRuns: make([]ReportOutputTestRun, 0),
	}
	for _, testRunResultOutput := range testRunResultOutputs {
		if testRunResultOutput.TestCaseId == 0 {
			continue
		}
		output.TestRuns = append(output.TestRuns, ReportOutputTestRun{
			TestCaseId: testRunResultOutput.TestCaseId,
			Status:     testRunResultOutput.Status,
		})
	}
	return
}

func printOutput(output ReportOutput) {
	jsonOutput, err := json.Marshal(output)
	if err != nil {
		log.Fatalf("Failed to marshal output: %v", err)
	}
	fmt.Println(string(jsonOutput))
}
