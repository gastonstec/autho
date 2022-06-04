package others

import	"encoding/xml"

// Request structs
type Req struct {
	XMLName    xml.Name     		`xml:"methodCall"`
	MethodName string       		`xml:"methodName"`
	TagParams  ReqParamsTag 		`xml:"params"`
}

type ReqParamsTag struct {
	TagParam []ReqParam 			`xml:"param"`
}

type ReqParam struct {
	TagValue ValueStr 				`xml:"value"`
}

type ValueStr struct {
	Value string 					`xml:",any"`
}

