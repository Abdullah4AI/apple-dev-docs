package workflow

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func newTestDefinition() *Definition {
	return &Definition{
		Env: map[string]string{"GLOBAL": "g"},
		Workflows: map[string]Workflow{
			"main": {
				Env:   map[string]string{"LOCAL": "l"},
				Steps: []Step{{Run: "echo hello"}},
			},
		},
	}
}

func runOpts(name string) RunOptions {
	return RunOptions{
		WorkflowName: name,
		Stdout:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
	}
}

func TestRun_SimpleEcho(t *testing.T) {
	def := newTestDefinition()
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected status ok, got %q", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != "ok" {
		t.Fatalf("expected step status ok, got %q", result.Steps[0].Status)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "hello") {
		t.Fatalf("expected stdout to contain 'hello', got %q", stdout)
	}
}

func TestRun_DryRun(t *testing.T) {
	def := newTestDefinition()
	opts := runOpts("main")
	opts.DryRun = true

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected status ok, got %q", result.Status)
	}
	if result.Steps[0].Status != "dry-run" {
		t.Fatalf("expected step status dry-run, got %q", result.Steps[0].Status)
	}
	// Verify echo was not actually executed
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if strings.Contains(stdout, "hello") {
		t.Fatal("dry-run should not execute commands")
	}
	// Verify dry-run preview on stderr
	stderr := opts.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "[dry-run]") {
		t.Fatalf("expected stderr to contain [dry-run], got %q", stderr)
	}
}

func TestRun_DryRunShowsRawCommand(t *testing.T) {
	def := &Definition{
		Env: map[string]string{"NAME": "world"},
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo $NAME"}}},
		},
	}
	opts := runOpts("test")
	opts.DryRun = true

	_, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stderr := opts.Stderr.(*bytes.Buffer).String()
	// Dry-run must NOT expand env values (avoids leaking secrets).
	if strings.Contains(stderr, "echo world") {
		t.Fatalf("dry-run must not expand env vars, got %q", stderr)
	}
	if !strings.Contains(stderr, "echo $NAME") {
		t.Fatalf("expected raw command in dry-run, got %q", stderr)
	}
}

func TestRun_ConditionalTruthy(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {
				Env:   map[string]string{"DO_IT": "true"},
				Steps: []Step{{Run: "echo yes", If: "DO_IT"}},
			},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Steps[0].Status != "ok" {
		t.Fatalf("expected status ok, got %q", result.Steps[0].Status)
	}
}

func TestRun_ConditionalFalsy(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo no", If: "MISSING_VAR"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Steps[0].Status != "skipped" {
		t.Fatalf("expected status skipped, got %q", result.Steps[0].Status)
	}
}

func TestRun_SkippedWorkflowStepIncludesWorkflowName(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{
				{Workflow: "helper", If: "SKIP_ME"},
				{Run: "echo done"},
			}},
			"helper": {Steps: []Step{{Run: "echo from-helper"}}},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps (skipped + done), got %d", len(result.Steps))
	}
	skipped := result.Steps[0]
	if skipped.Status != "skipped" {
		t.Fatalf("expected first step skipped, got %q", skipped.Status)
	}
	if skipped.Workflow != "helper" {
		t.Fatalf("expected skipped step to include workflow='helper', got %q", skipped.Workflow)
	}
}

func TestRun_SkippedRunStepIncludesCommand(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{
				{Run: "echo guarded", If: "NOPE"},
			}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	skipped := result.Steps[0]
	if skipped.Status != "skipped" {
		t.Fatalf("expected skipped, got %q", skipped.Status)
	}
	if skipped.Command != "echo guarded" {
		t.Fatalf("expected skipped step to include command='echo guarded', got %q", skipped.Command)
	}
}

func TestRun_ConditionalFalsyValues(t *testing.T) {
	values := []string{"0", "false", "no", "off"}
	for _, val := range values {
		t.Run(val, func(t *testing.T) {
			def := &Definition{
				Workflows: map[string]Workflow{
					"test": {
						Env:   map[string]string{"FLAG": val},
						Steps: []Step{{Run: "echo no", If: "FLAG"}},
					},
				},
			}
			opts := runOpts("test")
			result, err := Run(context.Background(), def, opts)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if result.Steps[0].Status != "skipped" {
				t.Fatalf("expected skipped for %q, got %q", val, result.Steps[0].Status)
			}
		})
	}
}

func TestRun_IfChecksOsEnv(t *testing.T) {
	t.Setenv("WORKFLOW_TEST_IF_VAR", "true")
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo yes", If: "WORKFLOW_TEST_IF_VAR"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Steps[0].Status != "ok" {
		t.Fatalf("expected ok (from os env), got %q", result.Steps[0].Status)
	}
}

func TestRun_EnvMerging(t *testing.T) {
	def := &Definition{
		Env: map[string]string{"VAR": "global"},
		Workflows: map[string]Workflow{
			"test": {
				Env:   map[string]string{"VAR": "local"},
				Steps: []Step{{Run: "echo $VAR"}},
			},
		},
	}
	opts := runOpts("test")
	opts.Params = map[string]string{"VAR": "param"}

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "param") {
		t.Fatalf("expected params to override, got %q", stdout)
	}
	if result.Steps[0].Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Steps[0].Status)
	}
}

func TestRun_RuntimeParamAffectsOutput(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo GREETING_IS_$GREETING"}}},
		},
	}
	opts := runOpts("test")
	opts.Params = map[string]string{"GREETING": "hello-world"}

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "GREETING_IS_hello-world") {
		t.Fatalf("expected runtime param in output, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_RuntimeParamControlsConditional(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{
				{Run: "echo skipped-step", If: "SHOULD_RUN"},
				{Run: "echo always-runs"},
			}},
		},
	}

	// Without the param — step skipped
	opts1 := runOpts("test")
	result1, err := Run(context.Background(), def, opts1)
	if err != nil {
		t.Fatalf("Run without param: %v", err)
	}
	if result1.Steps[0].Status != "skipped" {
		t.Fatalf("expected skipped without param, got %q", result1.Steps[0].Status)
	}

	// With the param — step runs
	opts2 := runOpts("test")
	opts2.Params = map[string]string{"SHOULD_RUN": "true"}
	result2, err := Run(context.Background(), def, opts2)
	if err != nil {
		t.Fatalf("Run with param: %v", err)
	}
	if result2.Steps[0].Status != "ok" {
		t.Fatalf("expected ok with param, got %q", result2.Steps[0].Status)
	}
	stdout := opts2.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "skipped-step") {
		t.Fatalf("expected conditional step to run, got %q", stdout)
	}
}

func TestRun_SubWorkflow(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{
				{Workflow: "helper"},
			}},
			"helper": {Steps: []Step{{Run: "echo from-helper"}}},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
	// Flattened: helper step has ParentWorkflow set
	found := false
	for _, s := range result.Steps {
		if s.ParentWorkflow == "helper" && s.Status == "ok" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected flattened sub-workflow step with ParentWorkflow=helper, got %v", result.Steps)
	}
}

func TestRun_SubWorkflowWithEnv(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{
				{Workflow: "helper", With: map[string]string{"MSG": "hello"}},
			}},
			"helper": {Steps: []Step{{Run: "echo $MSG"}}},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "hello") {
		t.Fatalf("expected 'hello' from with env, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_SubWorkflowWithEnv_WithOverridesSubWorkflowEnv(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{
				{Workflow: "helper", With: map[string]string{"MSG": "from-with"}},
			}},
			"helper": {
				Env:   map[string]string{"MSG": "from-helper-env"},
				Steps: []Step{{Run: "echo $MSG"}},
			},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "from-with") {
		t.Fatalf("expected with env to override sub-workflow env, got %q", stdout)
	}
	if strings.Contains(stdout, "from-helper-env") {
		t.Fatalf("expected sub-workflow env to be overridden, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_SubWorkflow_CallerEnvOverridesSubWorkflowEnv(t *testing.T) {
	// CLI params (via caller env) should beat sub-workflow env defaults.
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{
				{Workflow: "helper"},
			}},
			"helper": {
				Env:   map[string]string{"MSG": "from-helper-env"},
				Steps: []Step{{Run: "echo $MSG"}},
			},
		},
	}
	opts := runOpts("main")
	opts.Params = map[string]string{"MSG": "from-cli-param"}

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "from-cli-param") {
		t.Fatalf("expected CLI params to override sub-workflow env, got %q", stdout)
	}
	if strings.Contains(stdout, "from-helper-env") {
		t.Fatalf("sub-workflow env should not override CLI params, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_SubWorkflowEnvDoesNotLeak(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{
				{Workflow: "helper", With: map[string]string{"LEAKED": "yes"}},
				{Run: "echo LEAKED_IS_${LEAKED:-empty}"},
			}},
			"helper": {Steps: []Step{{Run: "echo $LEAKED"}}},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	// Helper sees LEAKED=yes
	if !strings.Contains(stdout, "yes") {
		t.Fatalf("helper should see LEAKED=yes, got %q", stdout)
	}
	// Parent's second step should not see LEAKED
	if strings.Contains(stdout, "LEAKED_IS_yes") {
		t.Fatal("LEAKED should not be visible in parent workflow")
	}
	if !strings.Contains(stdout, "LEAKED_IS_empty") {
		t.Fatalf("expected LEAKED_IS_empty in parent, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_PrivateWorkflow_DirectRunFails(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"secret": {
				Private: true,
				Steps:   []Step{{Run: "echo secret"}},
			},
		},
	}
	opts := runOpts("secret")

	_, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error for private workflow")
	}
	if !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected private error, got %v", err)
	}
}

func TestRun_PrivateWorkflow_SubWorkflowCallWorks(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{{Workflow: "secret"}}},
			"secret": {
				Private: true,
				Steps:   []Step{{Run: "echo secret"}},
			},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_StepFailure(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{
				{Run: "echo before"},
				{Run: "exit 1", Name: "failing"},
				{Run: "echo after"},
			}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error on step failure")
	}
	if result.Status != "error" {
		t.Fatalf("expected status error, got %q", result.Status)
	}
	if result.Error == "" || !strings.Contains(result.Error, "step 2") {
		t.Fatalf("expected result.Error to mention step 2, got %q", result.Error)
	}
	// Partial results: first step ok, second error, third not reached
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps (partial), got %d", len(result.Steps))
	}
	if result.Steps[0].Status != "ok" {
		t.Fatalf("expected first step ok, got %q", result.Steps[0].Status)
	}
	if result.Steps[1].Status != "error" {
		t.Fatalf("expected second step error, got %q", result.Steps[1].Status)
	}
}

func TestRun_AfterAllDoesNotRunOnStepFailure(t *testing.T) {
	def := &Definition{
		AfterAll: "echo after_all_should_not_run",
		Error:    "echo error_hook_ran",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "exit 1"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error on step failure")
	}
	if result.Status != "error" {
		t.Fatalf("expected status error, got %q", result.Status)
	}
	if result.Hooks == nil || result.Hooks.Error == nil {
		t.Fatalf("expected error hook to be recorded, got %+v", result.Hooks)
	}
	if result.Hooks.AfterAll != nil {
		t.Fatalf("expected after_all hook to not run on step failure, got %+v", result.Hooks.AfterAll)
	}

	stdout := opts.Stdout.(*bytes.Buffer).String()
	if strings.Contains(stdout, "after_all_should_not_run") {
		t.Fatalf("after_all hook should not have executed on step failure, got stdout %q", stdout)
	}
	if !strings.Contains(stdout, "error_hook_ran") {
		t.Fatalf("expected error hook output, got stdout %q", stdout)
	}
}

func TestRun_PipelineFailure(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{
				{Run: "false | cat", Name: "pipeline"},
			}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error on pipeline failure")
	}
	if result.Status != "error" {
		t.Fatalf("expected status error, got %q", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != "error" {
		t.Fatalf("expected step status error, got %q", result.Steps[0].Status)
	}
}

func TestRun_MaxCallDepthExceeded(t *testing.T) {
	// Create a chain that exceeds MaxCallDepth
	// We can't actually create a cycle (validation would catch it),
	// so we create a long chain
	workflows := make(map[string]Workflow)
	for i := 0; i <= MaxCallDepth+1; i++ {
		name := "w" + strings.Repeat("x", i)
		nextName := "w" + strings.Repeat("x", i+1)
		if i > MaxCallDepth {
			workflows[name] = Workflow{Steps: []Step{{Run: "echo done"}}}
		} else {
			workflows[name] = Workflow{Steps: []Step{{Workflow: nextName}}}
		}
	}
	def := &Definition{Workflows: workflows}
	opts := runOpts("w")

	_, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error for max call depth")
	}
	if !strings.Contains(err.Error(), "max call depth") {
		t.Fatalf("expected max call depth error, got %v", err)
	}
}

func TestRun_ErrorHook_DryRunDoesNotExecute(t *testing.T) {
	workflows := make(map[string]Workflow)
	for i := 0; i <= MaxCallDepth+1; i++ {
		name := "w" + strings.Repeat("x", i)
		nextName := "w" + strings.Repeat("x", i+1)
		if i > MaxCallDepth {
			workflows[name] = Workflow{Steps: []Step{{Run: "echo done"}}}
		} else {
			workflows[name] = Workflow{Steps: []Step{{Workflow: nextName}}}
		}
	}
	def := &Definition{
		Error:     "echo error_hook_ran",
		Workflows: workflows,
	}
	opts := runOpts("w")
	opts.DryRun = true

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error for max call depth")
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}

	// Dry-run should never execute hooks (only preview them).
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if strings.Contains(stdout, "error_hook_ran") {
		t.Fatalf("expected error hook to not execute in dry-run, got stdout %q", stdout)
	}
	stderr := opts.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "[dry-run] hook:") || !strings.Contains(stderr, "error_hook_ran") {
		t.Fatalf("expected dry-run hook preview in stderr, got %q", stderr)
	}
}

func TestRun_BeforeAllHook(t *testing.T) {
	def := &Definition{
		BeforeAll: "echo before_all",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo main"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "before_all") {
		t.Fatalf("expected before_all output, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_BeforeAllHookFailure(t *testing.T) {
	def := &Definition{
		BeforeAll: "exit 1",
		Error:     "echo error_hook_ran",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo should-not-run"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error on before_all failure")
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
	if result.Error == "" || !strings.Contains(result.Error, "before_all") {
		t.Fatalf("expected result.Error to mention before_all, got %q", result.Error)
	}
	if result.Hooks == nil || result.Hooks.BeforeAll == nil {
		t.Fatal("expected hooks.before_all to be recorded")
	}
	if result.Hooks.BeforeAll.Status != "error" {
		t.Fatalf("expected before_all hook status=error, got %q", result.Hooks.BeforeAll.Status)
	}
	if result.Hooks.BeforeAll.Error == "" {
		t.Fatal("expected before_all hook error detail")
	}
	if result.Hooks.Error == nil || result.Hooks.Error.Status != "ok" {
		t.Fatalf("expected error hook status=ok, got %+v", result.Hooks.Error)
	}
	// Steps should not have run
	if len(result.Steps) != 0 {
		t.Fatalf("expected 0 steps, got %d", len(result.Steps))
	}
	// Error hook should have fired
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "error_hook_ran") {
		t.Fatalf("expected error hook output, got %q", stdout)
	}
}

func TestRun_AfterAllHook(t *testing.T) {
	def := &Definition{
		AfterAll: "echo after_all",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo main"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "after_all") {
		t.Fatalf("expected after_all output, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_ErrorHook(t *testing.T) {
	def := &Definition{
		Error: "echo error_hook_ran",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "exit 1"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if result.Error == "" {
		t.Fatal("expected result.Error to be populated on failure")
	}
	if result.Hooks == nil || result.Hooks.Error == nil || result.Hooks.Error.Status != "ok" {
		t.Fatalf("expected hooks.error status=ok, got %+v", result.Hooks)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "error_hook_ran") {
		t.Fatalf("expected error hook output, got %q", stdout)
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
}

func TestRun_HooksDryRun(t *testing.T) {
	def := &Definition{
		BeforeAll: "echo should-not-execute",
		AfterAll:  "echo should-not-execute",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo hello"}}},
		},
	}
	opts := runOpts("test")
	opts.DryRun = true

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if strings.Contains(stdout, "should-not-execute") {
		t.Fatal("hooks should not execute in dry-run mode")
	}
	stderr := opts.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "[dry-run] hook:") {
		t.Fatalf("expected dry-run hook preview, got stderr %q", stderr)
	}
	if result.Hooks == nil || result.Hooks.BeforeAll == nil || result.Hooks.AfterAll == nil {
		t.Fatalf("expected before_all/after_all hooks to be recorded in dry-run, got %+v", result.Hooks)
	}
	if result.Hooks.BeforeAll.Status != "dry-run" || result.Hooks.AfterAll.Status != "dry-run" {
		t.Fatalf("expected hooks to have status=dry-run, got before_all=%+v after_all=%+v", result.Hooks.BeforeAll, result.Hooks.AfterAll)
	}
	if result.Hooks.Error != nil {
		t.Fatalf("expected hooks.error to be nil on success, got %+v", result.Hooks.Error)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "sleep 10"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(ctx, def, opts)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
}

func TestRun_UnknownWorkflow(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{{Run: "echo hi"}}},
		},
	}
	opts := runOpts("nonexistent")

	_, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error for unknown workflow")
	}
	if !strings.Contains(err.Error(), "unknown workflow") {
		t.Fatalf("expected unknown workflow error, got %v", err)
	}
}

func TestRun_AfterAllHookFailure(t *testing.T) {
	def := &Definition{
		AfterAll: "exit 1",
		Error:    "echo error_hook_fired",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo main"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error on after_all failure")
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
	if result.Error == "" || !strings.Contains(result.Error, "after_all") {
		t.Fatalf("expected result.Error to mention after_all, got %q", result.Error)
	}
	if result.Hooks == nil || result.Hooks.AfterAll == nil {
		t.Fatal("expected hooks.after_all to be recorded")
	}
	if result.Hooks.AfterAll.Status != "error" {
		t.Fatalf("expected after_all hook status=error, got %q", result.Hooks.AfterAll.Status)
	}
	if result.Hooks.AfterAll.Error == "" {
		t.Fatal("expected after_all hook error detail")
	}
	if result.Hooks.Error == nil || result.Hooks.Error.Status != "ok" {
		t.Fatalf("expected error hook status=ok, got %+v", result.Hooks.Error)
	}
	if !strings.Contains(err.Error(), "after_all") {
		t.Fatalf("expected after_all in error, got %v", err)
	}
	// Steps should have completed before hook failed
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != "ok" {
		t.Fatalf("expected step ok, got %q", result.Steps[0].Status)
	}
	// Error hook should fire on after_all failure
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "error_hook_fired") {
		t.Fatalf("expected error hook to fire on after_all failure, got %q", stdout)
	}
}

func TestRun_DryRunSubWorkflow(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"main":   {Steps: []Step{{Workflow: "helper"}}},
			"helper": {Steps: []Step{{Run: "echo from-helper"}}},
		},
	}
	opts := runOpts("main")
	opts.DryRun = true

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
	stderr := opts.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "[dry-run] step 1: workflow helper") {
		t.Fatalf("expected dry-run sub-workflow preview, got %q", stderr)
	}
	// Helper step should also be dry-run
	if !strings.Contains(stderr, "[dry-run] step 1: echo from-helper") {
		t.Fatalf("expected dry-run helper step preview, got %q", stderr)
	}
	// Should not actually execute
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if strings.Contains(stdout, "from-helper") {
		t.Fatal("dry-run should not execute sub-workflow commands")
	}
}

func TestRun_RuntimeUnknownSubWorkflow(t *testing.T) {
	// Bypass validation — directly construct a definition with a bad reference
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {Steps: []Step{{Workflow: "nonexistent"}}},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error for unknown sub-workflow at runtime")
	}
	if !strings.Contains(err.Error(), "unknown workflow") {
		t.Fatalf("expected unknown workflow error, got %v", err)
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
	// The failing step should be recorded
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != "error" {
		t.Fatalf("expected step error, got %q", result.Steps[0].Status)
	}
}

func TestRun_NilStdoutStderr(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo hello"}}},
		},
	}
	opts := RunOptions{
		WorkflowName: "test",
		// Stdout and Stderr are nil — should default to os.Stdout/os.Stderr
	}

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_MaxCallDepth_ExactLimit(t *testing.T) {
	// Create a chain of exactly MaxCallDepth levels — should succeed.
	// depth=0 is the entry workflow, so we need MaxCallDepth workflows
	// chained together to hit depth MaxCallDepth-1 (the last allowed).
	workflows := make(map[string]Workflow)
	for i := 0; i < MaxCallDepth; i++ {
		name := fmt.Sprintf("w%d", i)
		nextName := fmt.Sprintf("w%d", i+1)
		if i == MaxCallDepth-1 {
			workflows[name] = Workflow{Steps: []Step{{Run: "echo leaf"}}}
		} else {
			workflows[name] = Workflow{Steps: []Step{{Workflow: nextName}}}
		}
	}
	def := &Definition{Workflows: workflows}
	opts := runOpts("w0")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("expected success at exact max depth, got: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "leaf") {
		t.Fatalf("expected leaf output, got %q", stdout)
	}
}

func TestRun_ErrorHookFailure_OriginalErrorReturned(t *testing.T) {
	// When the error hook itself fails, the original step error should
	// still be returned (error hook failure is silently swallowed).
	def := &Definition{
		Error: "exit 42",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo fail && exit 1"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	// The error should be about the step failure, not the error hook failure.
	if strings.Contains(err.Error(), "exit status 42") {
		t.Fatalf("expected original step error, not error hook failure; got %v", err)
	}
	if !strings.Contains(err.Error(), "step 1") {
		t.Fatalf("expected step 1 error, got %v", err)
	}
	// But the error hook failure should be recorded in the structured result.
	if result.Hooks == nil || result.Hooks.Error == nil || result.Hooks.Error.Status != "error" {
		t.Fatalf("expected hooks.error status=error, got %+v", result.Hooks)
	}
	if !strings.Contains(result.Hooks.Error.Error, "exit status 42") {
		t.Fatalf("expected error hook detail to mention exit status 42, got %q", result.Hooks.Error.Error)
	}
}

func TestRun_SubWorkflow_EnvIsolation(t *testing.T) {
	// "with" overrides should not leak back to parent.
	// Parent sets X=parent, call-site overrides X=child via with.
	// A step after the sub-workflow call should see X=parent.
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {
				Env: map[string]string{"X": "parent"},
				Steps: []Step{
					{Workflow: "child", With: map[string]string{"X": "child"}},
					{Run: "echo X_IS_$X"},
				},
			},
			"child": {
				Steps: []Step{{Run: "echo CHILD_X_IS_$X"}},
			},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	// Child should see X=child (from with: override)
	if !strings.Contains(stdout, "CHILD_X_IS_child") {
		t.Fatalf("expected child to see X=child via with, got %q", stdout)
	}
	// Parent step after sub-workflow should still see X=parent
	if !strings.Contains(stdout, "X_IS_parent") {
		t.Fatalf("expected parent to see X=parent after sub-workflow, got %q", stdout)
	}
}

func TestRun_SubWorkflow_EnvDefaultsVsCallerOverride(t *testing.T) {
	// Sub-workflow env provides defaults; caller env should win.
	def := &Definition{
		Workflows: map[string]Workflow{
			"main": {
				Env: map[string]string{"X": "caller"},
				Steps: []Step{
					{Workflow: "child"},
				},
			},
			"child": {
				Env:   map[string]string{"X": "default", "Y": "child-only"},
				Steps: []Step{{Run: "echo X_IS_$X Y_IS_$Y"}},
			},
		},
	}
	opts := runOpts("main")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	// X: caller env should win over child default
	if !strings.Contains(stdout, "X_IS_caller") {
		t.Fatalf("expected caller env to override sub-workflow default, got %q", stdout)
	}
	// Y: child default should apply since caller doesn't set it
	if !strings.Contains(stdout, "Y_IS_child-only") {
		t.Fatalf("expected sub-workflow default for Y, got %q", stdout)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
}

func TestRun_DurationMS_Populated(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo fast"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.DurationMS < 0 {
		t.Fatalf("expected non-negative DurationMS, got %d", result.DurationMS)
	}
	if len(result.Steps) == 0 {
		t.Fatal("expected at least one step")
	}
	if result.Steps[0].DurationMS < 0 {
		t.Fatalf("expected non-negative step DurationMS, got %d", result.Steps[0].DurationMS)
	}
}

func TestRun_DurationMS_IncludesAfterAll(t *testing.T) {
	// after_all hook sleeps 100ms. DurationMS must include that time.
	def := &Definition{
		AfterAll: "sleep 0.1",
		Workflows: map[string]Workflow{
			"test": {Steps: []Step{{Run: "echo fast"}}},
		},
	}
	opts := runOpts("test")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok, got %q", result.Status)
	}
	// The after_all hook sleeps 100ms, so total duration must be >= 100ms.
	if result.DurationMS < 100 {
		t.Fatalf("expected DurationMS >= 100 (must include after_all time), got %d", result.DurationMS)
	}
}
