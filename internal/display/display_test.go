package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrint(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Print("hello", " ", "world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("Print output = %q, want to contain %q", buf.String(), "hello world")
	}
}

func TestPrintln(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Println("hello world")
	output := buf.String()
	if !strings.Contains(output, "hello world") {
		t.Errorf("Println output = %q, want to contain %q", output, "hello world")
	}
	if !strings.HasSuffix(output, "\n") {
		t.Error("Println output should end with newline")
	}
}

func TestPrintf(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Printf("hello %s", "world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("Printf output = %q, want to contain %q", buf.String(), "hello world")
	}
}

func TestSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Success("operation completed")
	output := buf.String()
	if !strings.Contains(output, "operation completed") {
		t.Errorf("Success output = %q, want to contain %q", output, "operation completed")
	}
	if !strings.Contains(output, "✓") {
		t.Error("Success output should contain checkmark icon")
	}
}

func TestError(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Error("operation failed")
	output := buf.String()
	if !strings.Contains(output, "operation failed") {
		t.Errorf("Error output = %q, want to contain %q", output, "operation failed")
	}
	if !strings.Contains(output, "✗") {
		t.Error("Error output should contain X icon")
	}
}

func TestWarning(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Warning("be careful")
	output := buf.String()
	if !strings.Contains(output, "be careful") {
		t.Errorf("Warning output = %q, want to contain %q", output, "be careful")
	}
	if !strings.Contains(output, "⚠") {
		t.Error("Warning output should contain warning icon")
	}
}

func TestInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Info("information message")
	output := buf.String()
	if !strings.Contains(output, "information message") {
		t.Errorf("Info output = %q, want to contain %q", output, "information message")
	}
	if !strings.Contains(output, "ℹ") {
		t.Error("Info output should contain info icon")
	}
}

func TestSuccessf(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Successf("completed %d tasks", 5)
	output := buf.String()
	if !strings.Contains(output, "completed 5 tasks") {
		t.Errorf("Successf output = %q, want to contain %q", output, "completed 5 tasks")
	}
	if !strings.Contains(output, "✓") {
		t.Error("Successf output should contain checkmark icon")
	}
}

func TestErrorf(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Errorf("failed with code %d", 404)
	output := buf.String()
	if !strings.Contains(output, "failed with code 404") {
		t.Errorf("Errorf output = %q, want to contain %q", output, "failed with code 404")
	}
	if !strings.Contains(output, "✗") {
		t.Error("Errorf output should contain X icon")
	}
}

func TestWarningf(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Warningf("found %d issues", 3)
	output := buf.String()
	if !strings.Contains(output, "found 3 issues") {
		t.Errorf("Warningf output = %q, want to contain %q", output, "found 3 issues")
	}
	if !strings.Contains(output, "⚠") {
		t.Error("Warningf output should contain warning icon")
	}
}

func TestInfof(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	p.Infof("processing %s", "data")
	output := buf.String()
	if !strings.Contains(output, "processing data") {
		t.Errorf("Infof output = %q, want to contain %q", output, "processing data")
	}
	if !strings.Contains(output, "ℹ") {
		t.Error("Infof output should contain info icon")
	}
}

func TestBold(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	boldText := p.Bold("important")
	if boldText == "" {
		t.Error("Bold returned empty string")
	}
	// Note: we can't easily test the ANSI codes without making assumptions
	// about the color library implementation, so we just verify it returns something
}

func TestFaint(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	faintText := p.Faint("subtle")
	if faintText == "" {
		t.Error("Faint returned empty string")
	}
}

func TestSuccessText(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	successText := p.SuccessText("green")
	if successText == "" {
		t.Error("SuccessText returned empty string")
	}
}

func TestErrorText(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	errorText := p.ErrorText("red")
	if errorText == "" {
		t.Error("ErrorText returned empty string")
	}
}

func TestWarningText(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	warningText := p.WarningText("yellow")
	if warningText == "" {
		t.Error("WarningText returned empty string")
	}
}

func TestInfoText(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	infoText := p.InfoText("cyan")
	if infoText == "" {
		t.Error("InfoText returned empty string")
	}
}

func TestNewStderr(t *testing.T) {
	p := NewStderr()
	if p == nil {
		t.Error("NewStderr returned nil")
	}
}

func TestNewStdout(t *testing.T) {
	p := NewStdout()
	if p == nil {
		t.Error("NewStdout returned nil")
	}
}
