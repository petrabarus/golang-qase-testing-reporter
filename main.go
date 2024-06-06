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
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	qase "go.qase.io/client"
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
		Args:             cobra.ExactArgs(1),
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
	config.Filename = args[0]

	log.Printf("Config: %+v", config)
	ctx = context.Background()

	initQaseClient()
}

func initQaseClient() {
	configuration := qase.NewConfiguration()
	configuration.AddDefaultHeader("Token", config.QaseApiToken)
	qaseClient = *qase.NewAPIClient(configuration)
}

func RunCommand(cmd *cobra.Command, args []string) {
	var err error
	fmt.Println("Running go-qase-testing-reporter")
	results, err := processFile(config.Filename)
	if err != nil {
		log.Fatalf("Failed to process file: %v", err)
	}

	//	log.Printf("Results: %+v", results)
	id, err := createNewRun(results)
	if err != nil {
		log.Fatalf("Failed to create test run: %v", err)
	}

	err = createTestRunResults(id, results)
	if err != nil {
		log.Fatalf("Failed to create test run result: %v", err)
	}

	err = completeRun(id)
	if err != nil {
		log.Fatalf("Failed to complete test run: %v", err)
	}
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

func createTestRunResults(runId int32, results []ReportResult) (err error) {
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

	qaseId, err := parseQaseId(content.Test)
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

func parseQaseId(test string) (int, error) {
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
