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

package frames

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// frameImpl is a frame implementation
type frameImpl struct {
	labels  map[string]interface{}
	columns []Column
	indices []Column

	byName map[string]int // name -> index in columns
}

// NewFrame returns a new MapFrame
func NewFrame(columns []Column, indices []Column, labels map[string]interface{}) (Frame, error) {
	if err := checkEqualLen(columns, indices); err != nil {
		return nil, err
	}

	byName := make(map[string]int)
	for i, col := range columns {
		byName[col.Name()] = i
	}

	frame := &frameImpl{
		labels:  labels,
		columns: columns,
		byName:  byName,
		indices: indices,
	}

	return frame, nil
}

// NewFrameFromMap returns a new MapFrame from a map
func NewFrameFromMap(data map[string]interface{}) (Frame, error) {
	var (
		columns = make([]Column, len(data))
		i       = 0
		col     Column
		err     error
	)

	for name, values := range data {
		switch values.(type) {
		case []int:
			col, err = NewSliceColumn(name, values.([]int))
			if err != nil {
				return nil, errors.Wrap(err, "can't create int column")
			}
		case []float64:
			col, err = NewSliceColumn(name, values.([]float64))
			if err != nil {
				return nil, errors.Wrap(err, "can't create float column")
			}
		case []string:
			col, err = NewSliceColumn(name, values.([]string))
			if err != nil {
				return nil, errors.Wrap(err, "can't create string column")
			}
		case []time.Time:
			col, err = NewSliceColumn(name, values.([]time.Time))
			if err != nil {
				return nil, errors.Wrap(err, "can't create time column")
			}
		default:
			return nil, fmt.Errorf("unsupported data type - %T", values)
		}

		columns[i] = col
		i++
	}

	return NewFrame(columns, nil, nil)
}

// Names returns the column names
func (mf *frameImpl) Names() []string {
	names := make([]string, len(mf.columns))

	for i := 0; i < len(mf.columns); i++ {
		names[i] = mf.columns[i].Name() // TODO: Check if exists?
	}

	return names
}

// Indices returns the index columns
func (mf *frameImpl) Indices() []Column {
	return mf.indices
}

// Labels returns the Label set, nil if there's none
func (mf *frameImpl) Labels() map[string]interface{} {
	return mf.labels
}

// Len is the number of rows
func (mf *frameImpl) Len() int {
	if len(mf.columns) > 0 {
		return mf.columns[0].Len()
	}

	return 0
}

// Column gets a column by name
func (mf *frameImpl) Column(name string) (Column, error) {
	// TODO: We can speed it up by calculating once, but then we'll use more memory
	i, ok := mf.byName[name]
	if !ok {
		return nil, fmt.Errorf("column %q not found", name)
	}

	return mf.columns[i], nil
}

// Slice return a new Frame with is slice of the original
func (mf *frameImpl) Slice(start int, end int) (Frame, error) {
	if err := validateSlice(start, end, mf.Len()); err != nil {
		return nil, err
	}

	colSlices, err := sliceCols(mf.columns, start, end)
	if err != nil {
		return nil, err
	}

	indexSlices, err := sliceCols(mf.indices, start, end)
	if err != nil {
		return nil, err
	}

	return NewFrame(colSlices, indexSlices, mf.labels)
}

// ColumnMessage is a for encoding a column
type ColumnMessage struct {
	Slice *SliceColumnMessage `msgpack:"slice,omitempty"`
	Label *LabelColumnMessage `msgpack:"label,omitempty"`
}

// FrameMessage is over-the-wire frame data
type FrameMessage struct {
	Columns []ColumnMessage        `msgpack:"columns"`
	Indices []ColumnMessage        `msgpack:"indices,omitempty"`
	Labels  map[string]interface{} `msgpack:"labels,omitempty"`
}

// Marshal marshals to native type
func (mf *frameImpl) Marshal() (interface{}, error) {
	msg := &FrameMessage{
		Labels: mf.Labels(),
	}
	var err error

	msg.Columns, err = mf.marshalColumns(mf.columns)
	if err != nil {
		return nil, err
	}

	msg.Indices, err = mf.marshalColumns(mf.Indices())
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (mf *frameImpl) marshalColumns(columns []Column) ([]ColumnMessage, error) {
	if columns == nil {
		return nil, nil
	}

	messages := make([]ColumnMessage, len(columns))
	for i, col := range columns {
		colMsg, err := mf.marshalColumn(col)
		if err != nil {
			return nil, err
		}

		switch colMsg.(type) {
		case *SliceColumnMessage:
			messages[i].Slice = colMsg.(*SliceColumnMessage)
		case *LabelColumnMessage:
			messages[i].Label = colMsg.(*LabelColumnMessage)
		default:
			return nil, fmt.Errorf("unknown marshaled message type - %T", colMsg)
		}
	}

	return messages, nil
}

func (mf *frameImpl) marshalColumn(col Column) (interface{}, error) {
	marshaler, ok := col.(Marshaler)
	if !ok {
		return nil, fmt.Errorf("column %q is not Marshaler", col.Name())
	}

	msg, err := marshaler.Marshal()
	if err != nil {
		return nil, errors.Wrapf(err, "can't marshal %q", col.Name())
	}

	return msg, nil
}

func validateSlice(start int, end int, size int) error {
	if start < 0 || end < 0 {
		return fmt.Errorf("negative indexing not supported")
	}

	if end < start {
		return fmt.Errorf("end < start")
	}

	if start >= size {
		return fmt.Errorf("start out of bounds")
	}

	if end >= size {
		return fmt.Errorf("end out of bounds")
	}

	return nil
}

func checkEqualLen(columns []Column, indices []Column) error {
	size := -1
	for _, col := range columns {
		if size == -1 { // first column
			size = col.Len()
			continue
		}

		if colSize := col.Len(); colSize != size {
			return fmt.Errorf("%q column size mismatch (%d != %d)", col.Name(), colSize, size)
		}
	}

	for i, col := range indices {
		if colSize := col.Len(); colSize != size {
			return fmt.Errorf("index column %d size mismatch (%d != %d)", i, colSize, size)
		}
	}

	return nil
}

func sliceCols(columns []Column, start int, end int) ([]Column, error) {
	slices := make([]Column, len(columns))
	for i, col := range columns {
		slice, err := col.Slice(start, end)
		if err != nil {
			return nil, errors.Wrapf(err, "can't get slice from %q", col.Name())
		}

		slices[i] = slice
	}

	return slices, nil
}
