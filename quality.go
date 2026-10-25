// Я отделяю контракт качества от файлов и параметров командной строки.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)

const MaxInputBytes int64 = 8 * 1024 * 1024
const MaxRecords = 100000

var header = []string{"order_id", "ordered_on", "sku", "quantity", "unit_price_minor", "currency"}
var identifier = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,31}$`)
var positive = regexp.MustCompile(`^[1-9][0-9]*$`)
var nonnegative = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
var dateShape = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

type Total struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

type Report struct {
	Version     int     `json:"report_version"`
	InputSHA256 string  `json:"input_sha256"`
	InputBytes  int64   `json:"input_bytes"`
	Records     int     `json:"records"`
	Accepted    int     `json:"accepted"`
	Rejected    int     `json:"rejected"`
	MaxBadBP    int     `json:"max_bad_bp"`
	Gate        string  `json:"gate"`
	Reason      string  `json:"reason"`
	Totals      []Total `json:"totals"`
}

type Rejection struct {
	Record int      `json:"record_number"`
	Codes  []string `json:"codes"`
	SHA256 string   `json:"row_sha256"`
}

func numbers(row []string) (int64, int64, []string) {
	codes := []string{}
	quantity, qerr := strconv.ParseInt(row[3], 10, 64)
	price, perr := strconv.ParseInt(row[4], 10, 64)
	if !positive.MatchString(row[3]) || qerr != nil || quantity > 1000000 {
		codes = append(codes, "QUANTITY")
	}
	if !nonnegative.MatchString(row[4]) || perr != nil || price > 1000000000 {
		codes = append(codes, "PRICE")
	}
	return quantity, price, codes
}

func validate(row []string, seen map[string]bool) (int64, []string) {
	if len(row) != len(header) {
		return 0, []string{"ROW_WIDTH"}
	}
	for _, field := range row {
		if !utf8.ValidString(field) {
			return 0, []string{"UTF8"}
		}
	}
	codes := []string{}
	if !identifier.MatchString(row[0]) {
		codes = append(codes, "ORDER_ID")
	}
	if !identifier.MatchString(row[2]) {
		codes = append(codes, "SKU")
	}
	if _, err := time.Parse("2006-01-02", row[1]); err != nil || !dateShape.MatchString(row[1]) {
		codes = append(codes, "DATE")
	}
	quantity, price, numericCodes := numbers(row)
	codes = append(codes, numericCodes...)
	if row[5] != "RUB" && row[5] != "USD" && row[5] != "EUR" {
		codes = append(codes, "CURRENCY")
	}
	// Я резервирую ID только после остальных проверок: плохая строка не блокирует исправленную.
	if len(codes) == 0 && seen[row[0]] {
		codes = append(codes, "DUPLICATE")
	}
	if len(codes) != 0 {
		return 0, codes
	}
	return quantity * price, codes
}

// Process я вызываю только с временными outputs: фатальный отказ нельзя публиковать как полный отчёт.
func Process(input io.Reader, accepted io.Writer, rejects io.Writer, maxBadBP int) (Report, error) {
	report := Report{Version: 1, MaxBadBP: maxBadBP, Totals: []Total{}}
	if maxBadBP < 0 || maxBadBP > 10000 {
		return report, errors.New("BAD_THRESHOLD")
	}
	bounded := &io.LimitedReader{R: input, N: MaxInputBytes + 1}
	hash := sha256.New()
	reader := csv.NewReader(io.TeeReader(bounded, hash))
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
	first, err := reader.Read()
	if bounded.N == 0 {
		return report, errors.New("INPUT_TOO_LARGE")
	}
	if err != nil {
		return report, errors.New("HEADER_REQUIRED")
	}
	if len(first) != len(header) {
		return report, errors.New("HEADER_MISMATCH")
	}
	for i := range header {
		if first[i] != header[i] {
			return report, errors.New("HEADER_MISMATCH")
		}
	}
	writer := csv.NewWriter(accepted)
	if err := writer.Write(header); err != nil {
		return report, errors.New("OUTPUT_WRITE")
	}
	encoder := json.NewEncoder(rejects)
	seen := make(map[string]bool)
	totals := map[string]int64{}
	for {
		row, readErr := reader.Read()
		if bounded.N == 0 {
			return report, errors.New("INPUT_TOO_LARGE")
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return report, errors.New("CSV_STRUCTURE_OR_IO")
		}
		report.Records++
		if report.Records > MaxRecords {
			return report, errors.New("RECORD_LIMIT")
		}
		amount, codes := validate(row, seen)
		if len(codes) != 0 {
			// Я сохраняю байты даже плохого UTF-8: длина поля + байты не дают неоднозначной склейки.
			digest := sha256.New()
			var length [8]byte
			for _, field := range row {
				binary.BigEndian.PutUint64(length[:], uint64(len(field)))
				digest.Write(length[:])
				digest.Write([]byte(field))
			}
			if err := encoder.Encode(Rejection{report.Records, codes, hex.EncodeToString(digest.Sum(nil))}); err != nil {
				return report, errors.New("OUTPUT_WRITE")
			}
			report.Rejected++
			continue
		}
		if err := writer.Write(row); err != nil {
			return report, errors.New("OUTPUT_WRITE")
		}
		seen[row[0]] = true
		if totals[row[5]] > math.MaxInt64-amount {
			return report, errors.New("AMOUNT_OVERFLOW")
		}
		totals[row[5]] += amount
		report.Accepted++
	}
	writer.Flush()
	if writer.Error() != nil {
		return report, errors.New("OUTPUT_WRITE")
	}
	report.InputSHA256 = hex.EncodeToString(hash.Sum(nil))
	report.InputBytes = MaxInputBytes + 1 - bounded.N
	report.Gate, report.Reason = "pass", "WITHIN_THRESHOLD"
	if report.Records == 0 {
		report.Gate, report.Reason = "fail", "EMPTY_INPUT"
	} else if int64(report.Rejected)*10000 > int64(report.Records)*int64(maxBadBP) {
		report.Gate, report.Reason = "fail", "REJECT_RATE"
	}
	for _, currency := range []string{"EUR", "RUB", "USD"} {
		if amount, ok := totals[currency]; ok {
			report.Totals = append(report.Totals, Total{currency, amount})
		}
	}
	return report, nil
}
