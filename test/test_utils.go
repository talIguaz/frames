package test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/nuclio/logger"
	"github.com/stretchr/testify/suite"
	"github.com/v3io/frames"
	"github.com/v3io/frames/pb"
	v3io "github.com/v3io/v3io-go/pkg/dataplane"
)

type SuiteCreateFunc = func(frames.Client, v3io.Container, logger.Logger) suite.TestingSuite

func floatCol(t testing.TB, name string, size int) frames.Column {
	random := rand.New(rand.NewSource(time.Now().Unix()))
	floats := make([]float64, size)
	for i := range floats {
		floats[i] = random.Float64()
	}

	col, err := frames.NewSliceColumn(name, floats)
	if err != nil {
		t.Fatal(err)
	}

	return col
}

func stringCol(t testing.TB, name string, size int) frames.Column {
	strings := make([]string, size)
	for i := range strings {
		strings[i] = fmt.Sprintf("val-%d", i)
	}

	col, err := frames.NewSliceColumn(name, strings)
	if err != nil {
		t.Fatal(err)
	}
	return col
}

func boolCol(t testing.TB, name string, size int) frames.Column {
	bools := make([]bool, size)
	for i := range bools {
		bools[i] = true
	}

	col, err := frames.NewSliceColumn(name, bools)
	if err != nil {
		t.Fatal(err)
	}
	return col
}

func timeCol(t testing.TB, name string, size int) frames.Column {
	times := make([]time.Time, size)
	now := time.Now()
	for i := range times {
		times[i] = now.Add(time.Duration(i) * time.Hour)
	}

	col, err := frames.NewSliceColumn(name, times)
	if err != nil {
		t.Fatal(err)
	}
	return col
}

func initializeNullColumns(length int) []*pb.NullValuesMap {
	nullValues := make([]*pb.NullValuesMap, length)
	for i := 0; i < length; i++ {
		nullValues[i] = &pb.NullValuesMap{NullColumns: make(map[string]bool)}
	}
	return nullValues
}

func validateFramesAreEqual(s suite.Suite, frame1, frame2 frames.Frame) {
	// Check length
	s.Require().Equal(frame1.Len(), frame2.Len(), "frames length is different")

	// Check Indices
	frame1IndicesCount, frame2IndicesCount := len(frame1.Indices()), len(frame2.Indices())
	s.Require().Equal(frame1IndicesCount, frame2IndicesCount, "frames indices length is different")
	frame1IndicesNames, frame2IndicesNames := make([]string, frame1IndicesCount), make([]string, frame2IndicesCount)
	for i := 0; i < frame1IndicesCount; i++ {
		frame1IndicesNames[i] = frame1.Indices()[i].Name()
		frame2IndicesNames[i] = frame2.Indices()[i].Name()
	}
	s.Require().EqualValues(frame1IndicesNames, frame2IndicesNames, "frames index names are different")

	// Check columns
	s.Require().EqualValues(frame1.Names(), frame2.Names(), "frames column names are different")
	s.Require().EqualValues(iteratorToSlice(frame1.IterRows(true)),
		iteratorToSlice(frame2.IterRows(true)),
		"frames values mismatch")
}

func iteratorToSlice(iter frames.RowIterator) []map[string]interface{} {
	var response []map[string]interface{}
	for iter.Next() {
		response = append(response, iter.Row())
	}
	return response
}
