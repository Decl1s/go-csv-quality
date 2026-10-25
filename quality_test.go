package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const columns = "order_id,ordered_on,sku,quantity,unit_price_minor,currency\n"
const valid = "A001,2026-10-25,SKU-A,2,15000,RUB\n"

func inspect(t *testing.T, data string, threshold int) (Report, string, []Rejection) {
	t.Helper()
	var accepted, rejected bytes.Buffer
	report, err := Process(strings.NewReader(data), &accepted, &rejected, threshold)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&rejected)
	items := []Rejection{}
	for {
		var item Rejection
		err := decoder.Decode(&item)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, item)
	}
	return report, accepted.String(), items
}

func TestDemoContract(t *testing.T) {
	report, accepted, rejects := inspect(t, fixture, 0)
	if report.Records != 8 || report.Accepted != 4 || report.Rejected != 4 || report.Gate != "fail" {
		t.Fatal(report)
	}
	if len(rejects) != 4 || rejects[0].Record != 3 || rejects[1].Codes[0] != "DUPLICATE" {
		t.Fatal(rejects)
	}
	rows, err := csv.NewReader(strings.NewReader(accepted)).ReadAll()
	if err != nil || len(rows) != 5 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	want := []Total{{"EUR", 3000}, {"RUB", 39000}, {"USD", 2000}}
	for i := range want {
		if report.Totals[i] != want[i] {
			t.Fatal(report.Totals)
		}
	}
	digest := sha256.Sum256([]byte(fixture))
	if report.InputSHA256 != hex.EncodeToString(digest[:]) || report.InputBytes != int64(len(fixture)) {
		t.Fatal(report)
	}
}

func TestThresholdBoundary(t *testing.T) {
	for _, item := range []struct {
		threshold int
		gate      string
	}{{4999, "fail"}, {5000, "pass"}, {10000, "pass"}} {
		report, _, _ := inspect(t, fixture, item.threshold)
		if report.Gate != item.gate {
			t.Fatal(item, report)
		}
	}
}

func TestEmptyDataFailsGate(t *testing.T) {
	report, _, _ := inspect(t, columns, 10000)
	if report.Gate != "fail" || report.Reason != "EMPTY_INPUT" {
		t.Fatal(report)
	}
}

func TestHeadersAreExact(t *testing.T) {
	for _, data := range []string{"", "wrong\n", "\xef\xbb\xbf" + columns, "order_id,ordered_on,sku,quantity,unit_price_minor,currency,extra\n"} {
		if _, err := Process(strings.NewReader(data), io.Discard, io.Discard, 0); err == nil {
			t.Fatal("header accepted")
		}
	}
}

func TestContractRejections(t *testing.T) {
	for _, item := range []struct{ row, code string }{
		{"=cmd,2026-10-25,SKU-A,2,15000,RUB", "ORDER_ID"},
		{"A1,2026-10-25,=cmd,2,15000,RUB", "SKU"},
		{"A1,2026-02-30,SKU-A,2,15000,RUB", "DATE"},
		{"A1,2026-10-25,SKU-A,+2,15000,RUB", "QUANTITY"},
		{"A1,2026-10-25,SKU-A,02,15000,RUB", "QUANTITY"},
		{"A1,2026-10-25,SKU-A,2,15.00,RUB", "PRICE"},
		{"A1,2026-10-25,SKU-A,2,1000000001,RUB", "PRICE"},
		{"A1,2026-10-25,SKU-A,2,15000,BTC", "CURRENCY"},
		{"A1,too,few", "ROW_WIDTH"},
		{"A1,2026-10-25,\xff,2,15000,RUB", "UTF8"},
	} {
		t.Run(item.code+item.row[:2], func(t *testing.T) {
			report, _, rejects := inspect(t, columns+item.row+"\n", 10000)
			if report.Rejected != 1 || rejects[0].Codes[0] != item.code {
				t.Fatal(report, rejects)
			}
		})
	}
}

func TestRejectedIDDoesNotReserveKey(t *testing.T) {
	report, _, _ := inspect(t, columns+"A001,2026-02-30,SKU-A,2,15000,RUB\n"+valid+valid, 10000)
	if report.Accepted != 1 || report.Rejected != 2 {
		t.Fatal(report)
	}
}

func TestRejectionsDoNotContainSourceValues(t *testing.T) {
	var accepted, rejected bytes.Buffer
	_, err := Process(strings.NewReader(columns+"private-marker,2026-02-30,SKU-A,2,1,RUB\n"), &accepted, &rejected, 10000)
	if err != nil || strings.Contains(rejected.String(), "private-marker") {
		t.Fatal(err, rejected.String())
	}
}

func TestCSVQuotingFailureIsFatal(t *testing.T) {
	if _, err := Process(strings.NewReader(columns+valid+"\"broken\n"), io.Discard, io.Discard, 0); err == nil {
		t.Fatal("malformed CSV accepted")
	}
}

func TestInputLimit(t *testing.T) {
	if _, err := Process(strings.NewReader(columns+strings.Repeat("x", int(MaxInputBytes))), io.Discard, io.Discard, 0); err == nil || err.Error() != "INPUT_TOO_LARGE" {
		t.Fatal(err)
	}
}

func TestRecordLimit(t *testing.T) {
	if _, err := Process(strings.NewReader(columns+strings.Repeat("a,b\n", MaxRecords+1)), io.Discard, io.Discard, 10000); err == nil || err.Error() != "RECORD_LIMIT" {
		t.Fatal(err)
	}
}

func TestAmountOverflow(t *testing.T) {
	var data strings.Builder
	data.WriteString(columns)
	for i := 0; i < 10000; i++ {
		data.WriteString("A" + strconv.Itoa(i) + ",2026-10-25,SKU-A,1000000,1000000000,RUB\n")
	}
	if _, err := Process(strings.NewReader(data.String()), io.Discard, io.Discard, 0); err == nil || err.Error() != "AMOUNT_OVERFLOW" {
		t.Fatal(err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("sensitive-write-detail") }
func TestOutputFailureIsSanitized(t *testing.T) {
	_, err := Process(strings.NewReader(columns+valid), failingWriter{}, io.Discard, 0)
	if err == nil || err.Error() != "OUTPUT_WRITE" {
		t.Fatal(err)
	}
}

func TestPublishAndNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "in.csv"), filepath.Join(dir, "out")
	if err := os.WriteFile(input, []byte(fixture), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := publish(input, output, 5000)
	if err != nil || report.Gate != "pass" {
		t.Fatal(err, report)
	}
	before, _ := os.ReadFile(filepath.Join(output, "report.json"))
	if _, err := publish(input, output, 0); err == nil {
		t.Fatal("overwrote output")
	}
	after, _ := os.ReadFile(filepath.Join(output, "report.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("report changed")
	}
}

func TestFatalPublishLeavesNoResult(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "in.csv"), filepath.Join(dir, "out")
	os.WriteFile(input, []byte(columns+valid+"\"broken\n"), 0600)
	if _, err := publish(input, output, 0); err == nil {
		t.Fatal("publish succeeded")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatal("partial result published")
	}
}

func TestCommandExitCodes(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in.csv")
	var stdout, stderr bytes.Buffer
	if code := command([]string{"demo", "-file", input}, &stdout, &stderr); code != 0 {
		t.Fatal(code)
	}
	for i, threshold := range []string{"0", "5000"} {
		code := command([]string{"check", "-input", input, "-out", filepath.Join(dir, threshold), "-max-bad-bp", threshold}, &stdout, &stderr)
		if (i == 0 && code != 3) || (i == 1 && code != 0) {
			t.Fatal(code)
		}
	}
	if code := command([]string{"check", "-input", input, "-out", filepath.Join(dir, "x"), "-max-bad-bp", "10001"}, &stdout, &stderr); code != 2 {
		t.Fatal(code)
	}
}

func TestRowHashPreservesInvalidUTF8Bytes(t *testing.T) {
	_, _, first := inspect(t, columns+"A1,2026-10-25,\xff,1,1,RUB\n", 10000)
	_, _, second := inspect(t, columns+"A1,2026-10-25,\xfe,1,1,RUB\n", 10000)
	if first[0].SHA256 == second[0].SHA256 {
		t.Fatal("different invalid bytes collapsed")
	}
}

func TestExistingLockRefusesPublication(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "in.csv"), filepath.Join(dir, "out")
	os.WriteFile(input, []byte(fixture), 0600)
	os.WriteFile(output+".lock", []byte("owned-marker"), 0600)
	if _, err := publish(input, output, 5000); err == nil {
		t.Fatal("lock ignored")
	}
	marker, err := os.ReadFile(output + ".lock")
	if err != nil || string(marker) != "owned-marker" {
		t.Fatal("existing lock changed")
	}
}
