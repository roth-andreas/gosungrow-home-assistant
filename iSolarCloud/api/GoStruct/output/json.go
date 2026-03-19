package output

import (
	"encoding/json"
	"fmt"
)

type Json string

func (j Json) String() string {
	return string(j)
}

func GetAsJson(ref interface{}) Json {
	data, err := json.Marshal(ref)
	if err != nil {
		return Json(fmt.Sprintf(`{"error": %q}`, err.Error()))
	}
	return Json(data)
}

func GetAsPrettyJson(ref interface{}) Json {
	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return Json(fmt.Sprintf(`{"error": %q}`, err.Error()))
	}
	return Json(data)
}

func GetRequestString(ref interface{}) string {
	return GetAsPrettyJson(ref).String()
}

func GetEndPointString(ref interface{}) string {
	return GetAsPrettyJson(ref).String()
}
