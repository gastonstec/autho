
// Copyright Kueski. All rights reserved.
// Use of this source code is not licensed
// Package load handles the load transactions.
package load

import(
	"encoding/xml"
)

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