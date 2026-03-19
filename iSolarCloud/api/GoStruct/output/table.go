package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GraphRequest struct {
	Title       string
	SubTitle    string
	TimeColumn  *string
	DataColumn  *string
	UnitsColumn *string
	NameColumn  *string
	DataUnit    *string
	DataMin     *float64
	DataMax     *float64
	Width       *int
	Height      *int
	Error       error
}

type Tables map[string]Table

func NewTables() Tables {
	return make(Tables)
}

func (t Tables) Sort() []string {
	keys := make([]string, 0, len(t))
	for key := range t {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type Table struct {
	Name       string
	OutputType OutputType

	title      string
	directory  string
	filePrefix string
	json       []byte
	raw        []byte
	headers    []string
	rows       [][]string
	saveAsFile bool
	graph      *GraphRequest
}

func NewTable(headers ...string) Table {
	return Table{headers: append([]string{}, headers...)}
}

func (t *Table) SetName(name string) {
	t.Name = name
}

func (t *Table) SetTitle(format string, args ...interface{}) {
	t.title = fmt.Sprintf(format, args...)
}

func (t *Table) AppendTitle(format string, args ...interface{}) {
	if format == "" {
		return
	}
	t.title += fmt.Sprintf(format, args...)
}

func (t *Table) GetTitle() string {
	return t.title
}

func (t *Table) SetFilePrefix(format string, args ...interface{}) {
	t.filePrefix = sanitizeFilename(fmt.Sprintf(format, args...))
}

func (t *Table) PrependFilePrefix(prefix string) {
	prefix = sanitizeFilename(prefix)
	if prefix == "" {
		return
	}
	if t.filePrefix == "" {
		t.filePrefix = prefix
		return
	}
	t.filePrefix = prefix + "-" + t.filePrefix
}

func (t *Table) AppendFilePrefix(suffix string) {
	suffix = sanitizeFilename(suffix)
	if suffix == "" {
		return
	}
	if t.filePrefix == "" {
		t.filePrefix = suffix
		return
	}
	t.filePrefix += "-" + suffix
}

func (t *Table) GetFilePrefix() string {
	return t.filePrefix
}

func (t *Table) SetJson(data []byte) {
	if data == nil {
		t.json = nil
		return
	}
	t.json = append([]byte{}, data...)
}

func (t *Table) SetRaw(data []byte) {
	if data == nil {
		t.raw = nil
		return
	}
	t.raw = append([]byte{}, data...)
}

func (t *Table) SetGraphFilter(_ string) {}

func (t *Table) SetGraph(request GraphRequest) error {
	t.graph = &request
	return nil
}

func (t *Table) SetDirectory(directory string) {
	t.directory = directory
}

func (t *Table) SetSaveFile(yes bool) {
	t.saveAsFile = yes
}

func (t *Table) GetHeaders() []string {
	return append([]string{}, t.headers...)
}

func (t *Table) GetAllHeaders() []string {
	return t.GetHeaders()
}

func (t *Table) Height() int {
	return len(t.rows)
}

func (t *Table) AddRow(values ...interface{}) error {
	row := make([]string, 0, len(values))
	for _, value := range values {
		row = append(row, fmt.Sprint(value))
	}
	t.rows = append(t.rows, row)
	return nil
}

func (t *Table) Sort(column string) {
	if column == "" {
		return
	}
	index := -1
	for i, header := range t.headers {
		if header == column {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	sort.SliceStable(t.rows, func(i, j int) bool {
		left := ""
		right := ""
		if index < len(t.rows[i]) {
			left = t.rows[i][index]
		}
		if index < len(t.rows[j]) {
			right = t.rows[j][index]
		}
		return left < right
	})
}

func (t Table) String() string {
	lines := make([]string, 0, len(t.rows)+2)
	if t.title != "" {
		lines = append(lines, t.title)
	}
	if len(t.headers) > 0 {
		lines = append(lines, strings.Join(t.headers, "\t"))
	}
	for _, row := range t.rows {
		lines = append(lines, strings.Join(row, "\t"))
	}
	if len(lines) == 0 && len(t.raw) > 0 {
		return string(t.raw)
	}
	if len(lines) == 0 && len(t.json) > 0 {
		return string(t.json)
	}
	return strings.Join(lines, "\n")
}

func (t Table) Output() error {
	content := t.String()
	if !t.saveAsFile {
		if content != "" {
			fmt.Println(content)
		}
		return nil
	}

	filename := t.filePrefix
	if filename == "" {
		filename = sanitizeFilename(t.Name)
	}
	if filename == "" {
		filename = "gosungrow-output"
	}
	if !strings.Contains(filepath.Base(filename), ".") {
		filename += extensionForType(t.OutputType)
	}
	if t.directory != "" {
		filename = filepath.Join(t.directory, filename)
	}
	return os.WriteFile(filename, []byte(content), DefaultFileMode)
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	return value
}

func extensionForType(outputType OutputType) string {
	switch outputType {
	case StringTypeJson, StringTypeStruct:
		return ".json"
	case StringTypeCsv:
		return ".csv"
	case StringTypeXML:
		return ".xml"
	case StringTypeXLSX:
		return ".xlsx"
	default:
		return ".txt"
	}
}
