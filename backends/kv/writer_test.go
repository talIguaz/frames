/*
Copyright 2018 Iguazio Systems Ltd.

Licensed under the Apache License, Version 2.0 (the "License") with
an addition restriction as set forth herein. You may not use this
file except in compliance with the License. You may obtain a copy of
the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.

In addition, you may not use the software for any purposes that are
illegal under applicable law, and the grant of the foregoing license
under the Apache 2.0 license is conditioned upon your compliance with
such restriction.
*/

package kv

import (
	"fmt"
	"testing"
	"time"

	"github.com/v3io/frames"
	"github.com/v3io/frames/test"
)

func TestWriter(t *testing.T) {
	t.Skip("TODO")
}

func generateSequentialSampleFrameWithTypes(t *testing.T, size int, indexName string, columnNames map[string]string) frames.Frame {
	var icol frames.Column

	index := make([]int, size)
	for i := 0; i < size; i++ {
		index[i] = i
	}

	icol, err := frames.NewSliceColumn(indexName, index)
	if err != nil {
		t.Fatal(err)
	}

	columns := make([]frames.Column, len(columnNames))
	i := 0
	for columnName, columnType := range columnNames {
		switch columnType {
		case "string":
			columns[i] = test.StringCol(t, columnName, size)
		case "float":
			columns[i] = test.FloatCol(t, columnName, size)
		case "bool":
			columns[i] = test.BoolCol(t, columnName, size)
		case "time":
			columns[i] = test.TimeCol(t, columnName, size)
		default:
			t.Fatalf("type %v not supported", columnType)
		}

		i++
	}

	frame, err := frames.NewFrame(columns, []frames.Column{icol}, nil)
	if err != nil {
		t.Fatal(err)
	}

	return frame
}

func TestGenExpr(t *testing.T) {
	frame := generateSequentialSampleFrameWithTypes(t, 1,
		"idx", map[string]string{"n1": "float", "n2": "time", "n3": "string", "n4": "bool"})
	expression := "n1={n1};n2={n2};n3={n3};n4={n4};idx={idx};"

	frameData := test.FrameToDataMap(frame)["0"]

	actual, err := genExpr(expression, frame, 0)
	if err != nil {
		t.Fatalf("failed to generate expression, err: %v", err)
	}

	idx, n1, n2, n3, n4 := frameData["idx"], frameData["n1"], frameData["n2"], frameData["n3"], frameData["n4"]
	n2Time := n2.(time.Time)
	n2Seconds, n2Nanos := n2Time.Unix(), n2Time.Nanosecond()

	expected := fmt.Sprintf("n1=%v;n2=%v:%v;n3='%v';n4=%v;idx=%v;", n1, n2Seconds, n2Nanos, n3, n4, idx)

	if expected != actual {
		t.Fatalf("expression didn't match expected. \nexpected: %v \n actual: %v", expected, actual)
	}
}
