// Я оставляю в CLI только параметры, файловую транзакцию и exit codes.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const version = "0.1.0"
const fixture = `order_id,ordered_on,sku,quantity,unit_price_minor,currency
A001,2026-10-25,SKU-A,2,15000,RUB
A002,2026-10-25,SKU-B,1,9000,RUB
A003,2026-02-30,SKU-C,1,1000,RUB
A001,2026-10-25,SKU-A,2,15000,RUB
A004,2026-10-25,SKU-D,0,1000,RUB
A005,2026-10-25,SKU-E,1,2000,USD
A006,2026-10-25,SKU-F,1,1000,BTC
A007,2026-10-25,SKU-G,3,1000,EUR
`

func writeFile(path string, value []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("OUTPUT_CREATE")
	}
	_, writeErr := file.Write(value)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errors.New("OUTPUT_WRITE")
	}
	return nil
}

func publish(input, output string, maxBadBP int) (Report, error) {
	var report Report
	if maxBadBP < 0 || maxBadBP > 10000 {
		return report, errors.New("BAD_THRESHOLD")
	}
	file, err := os.Open(input)
	if err != nil {
		return report, errors.New("INPUT_OPEN")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return report, errors.New("INPUT_NOT_REGULAR")
	}
	output = filepath.Clean(output)
	parent := filepath.Dir(output)
	// Я ожидаю существующий parent: ошибочный путь не создаёт дерево каталогов молча.
	lock, err := os.OpenFile(output+".lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return report, errors.New("OUTPUT_LOCKED_OR_PARENT_MISSING")
	}
	lock.Close()
	defer os.Remove(output + ".lock")
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return report, errors.New("OUTPUT_EXISTS")
	}
	stage, err := os.MkdirTemp(parent, ".csv-quality-")
	if err != nil {
		return report, errors.New("STAGE_CREATE")
	}
	defer os.RemoveAll(stage)
	accepted, err := os.OpenFile(filepath.Join(stage, "accepted.csv"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return report, errors.New("OUTPUT_CREATE")
	}
	defer accepted.Close()
	rejects, err := os.OpenFile(filepath.Join(stage, "rejections.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return report, errors.New("OUTPUT_CREATE")
	}
	defer rejects.Close()
	report, err = Process(file, accepted, rejects, maxBadBP)
	if err != nil {
		return report, err
	}
	if err := accepted.Sync(); err != nil {
		return report, errors.New("OUTPUT_SYNC")
	}
	if err := rejects.Sync(); err != nil {
		return report, errors.New("OUTPUT_SYNC")
	}
	if err := accepted.Close(); err != nil {
		return report, errors.New("OUTPUT_CLOSE")
	}
	if err := rejects.Close(); err != nil {
		return report, errors.New("OUTPUT_CLOSE")
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return report, errors.New("REPORT_ENCODE")
	}
	if err := writeFile(filepath.Join(stage, "report.json"), append(data, '\n')); err != nil {
		return report, err
	}
	if err := os.Rename(stage, output); err != nil {
		return report, errors.New("PUBLISH_UNCONFIRMED")
	}
	return report, nil
}

func command(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Команды: demo, check, version")
		return 2
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Я не вывожу значения ошибочных аргументов, которые могут содержать данные.
	if args[0] == "demo" {
		file := flags.String("file", "demo.csv", "Новый файл синтетических заказов")
		if flags.Parse(args[1:]) != nil || flags.NArg() != 0 {
			fmt.Fprintln(stderr, "ARGUMENTS")
			return 2
		}
		if err := writeFile(*file, []byte(fixture)); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		fmt.Fprintln(stdout, "Я создал синтетический CSV: 8 строк, 4 корректные, 4 отклоняемые.")
		return 0
	}
	if args[0] != "check" {
		fmt.Fprintln(stderr, "COMMAND_UNKNOWN")
		return 2
	}
	input := flags.String("input", "", "Входной CSV")
	out := flags.String("out", "", "Новый каталог результатов")
	threshold := flags.Int("max-bad-bp", 0, "Допустимая доля брака в basis points, 0…10000")
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 || *input == "" || *out == "" {
		fmt.Fprintln(stderr, "ARGUMENTS")
		return 2
	}
	report, err := publish(*input, *out, *threshold)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		fmt.Fprintln(stderr, "STDOUT_WRITE")
		return 2
	}
	if report.Gate == "fail" {
		return 3
	}
	return 0
}

func main() { os.Exit(command(os.Args[1:], os.Stdout, os.Stderr)) }
