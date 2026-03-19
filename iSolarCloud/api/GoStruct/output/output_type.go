package output

type OutputType string

const (
	StringTypeNone     OutputType = ""
	StringTypeList     OutputType = "list"
	StringTypeTable    OutputType = "table"
	StringTypeRaw      OutputType = "raw"
	StringTypeJson     OutputType = "json"
	StringTypeCsv      OutputType = "csv"
	StringTypeGraph    OutputType = "graph"
	StringTypeXML      OutputType = "xml"
	StringTypeXLSX     OutputType = "xlsx"
	StringTypeMarkDown OutputType = "markdown"
	StringTypeStruct   OutputType = "struct"
)

func (o *OutputType) Set(value string) {
	*o = OutputType(value)
}

func (o *OutputType) SetNone()     { *o = StringTypeNone }
func (o *OutputType) SetList()     { *o = StringTypeList }
func (o *OutputType) SetTable()    { *o = StringTypeTable }
func (o *OutputType) SetRaw()      { *o = StringTypeRaw }
func (o *OutputType) SetJson()     { *o = StringTypeJson }
func (o *OutputType) SetCsv()      { *o = StringTypeCsv }
func (o *OutputType) SetGraph()    { *o = StringTypeGraph }
func (o *OutputType) SetXML()      { *o = StringTypeXML }
func (o *OutputType) SetXLSX()     { *o = StringTypeXLSX }
func (o *OutputType) SetMarkDown() { *o = StringTypeMarkDown }
func (o *OutputType) SetStruct()   { *o = StringTypeStruct }

func (o OutputType) IsNone() bool     { return o == StringTypeNone }
func (o OutputType) IsList() bool     { return o == StringTypeList }
func (o OutputType) IsTable() bool    { return o == StringTypeTable || o.IsNone() }
func (o OutputType) IsRaw() bool      { return o == StringTypeRaw }
func (o OutputType) IsJson() bool     { return o == StringTypeJson }
func (o OutputType) IsCsv() bool      { return o == StringTypeCsv }
func (o OutputType) IsGraph() bool    { return o == StringTypeGraph }
func (o OutputType) IsXML() bool      { return o == StringTypeXML }
func (o OutputType) IsXLSX() bool     { return o == StringTypeXLSX }
func (o OutputType) IsMarkDown() bool { return o == StringTypeMarkDown }
func (o OutputType) IsStruct() bool   { return o == StringTypeStruct }
