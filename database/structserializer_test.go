package database

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
)

func TestStructSerializer(t *testing.T) {

	dbe_ctx, dbe_conn, _ := ContextWithDb(context.Background(), nil, "postgres")
	defer ReleaseConn(dbe_ctx, dbe_conn)

	dbe.DeleteDatabase(dbe_ctx, "test")
	db, err := dbe.CreateActiveDatabase(dbe_ctx, "test", true)
	if err != nil {
		t.Fatal(err)
	}

	ctx, conn, _ := ContextWithDb(dbe_ctx, db, "postgres")
	defer ReleaseConn(ctx, conn)

	CreateTable(ctx, &Table{Name: "t1", Columns: []Column{
		{Name: "int2", Type: "int2"},
		{Name: "int4", Type: "int4"},
		{Name: "int8", Type: "int8"},
		{Name: "float4", Type: "float4"},
		{Name: "float8", Type: "float8"},
		{Name: "bool", Type: "bool"},
		{Name: "text", Type: "text"},
		{Name: "json", Type: "json"},
		{Name: "jsonb", Type: "jsonb"},
		{Name: "date", Type: "timestamp"},
	}})

	_, _, err = CreateRecords(ctx, "t1",
		[]Record{
			{
				"int2":   30_000,
				"int4":   200_000,
				"int8":   5_000_000_000,
				"float4": 3.14,
				"float8": 3.1415,
				"bool":   true,
				"text":   "smoothdb",
				"json":   "[123]",
				"jsonb":  "[\"test\"]",
				"date":   "2022-10-11T19:00",
			},
			{
				"int2":   nil,
				"int4":   nil,
				"int8":   nil,
				"float4": nil,
				"float8": nil,
				"bool":   nil,
				"text":   nil,
				"json":   nil,
				"jsonb":  nil,
				"date":   nil,
			},
			{
				"int2":   0,
				"int4":   0,
				"int8":   0,
				"float4": 0,
				"float8": 0,
				"bool":   false,
				"text":   "",
				"json":   "{}",
				"jsonb":  "{}",
				"date":   "0001-01-01T00:00",
			},
		},
		nil)
	if err != nil {
		t.Fatal(err)
	}

	gi := GetSmoothContext(ctx)
	rows, err := gi.Conn.Query(ctx, "select * from t1")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	structs, err := rowsToStructs(rows)
	if err != nil {
		t.Fatal(err)
	}

	type Struct struct {
		int2   int16
		b1     bool
		int4   int32
		b2     bool
		int8   int64
		b3     bool
		float4 float32
		b4     bool
		float8 float64
		b5     bool
		bool   bool
		b6     bool
		text   string
		b7     bool
		json   string
		b8     bool
		jsonb  string
		b9     bool
		date   time.Time
		b0     bool
	}

	date, _ := time.Parse(time.RFC3339, "2022-10-11T19:00:00Z")
	ss := Struct{
		30_000, false,
		200_000, false,
		5_000_000_000, false,
		3.14, false,
		3.1415, false,
		true, false,
		"smoothdb", false,
		"[123]", false,
		"[\"test\"]", false,
		date, false,
	}
	s := structs[0]

	if !compareStruct(s, ss) {
		t.Errorf("Expected \n\t\"%v\", \ngot \n\t\"%v\" \n", ss, s)
	}

	ss = Struct{
		0, true,
		0, true,
		0, true,
		0, true,
		0, true,
		false, true,
		"", true,
		"", true,
		"", true,
		time.Time{}, true,
	}
	s = structs[1]

	if !compareStruct(s, ss) {
		t.Errorf("Expected \n\t\"%v\", \ngot \n\t\"%v\" \n", ss, s)
	}

	ss = Struct{
		0, false,
		0, false,
		0, false,
		0, false,
		0, false,
		false, false,
		"", false,
		"{}", false,
		"{}", false,
		time.Time{}, false,
	}
	s = structs[2]

	if !compareStruct(s, ss) {
		t.Errorf("Expected \n\t\"%v\", \ngot \n\t\"%v\" \n", ss, s)
	}

	//

	rows, err = gi.Conn.Query(ctx, "select * from t1")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	structs, err = rowsToStructsWithPointers(rows)
	if err != nil {
		t.Fatal(err)
	}

	type StructPointers struct {
		int2   *int16
		int4   *int32
		int8   *int64
		float4 *float32
		float8 *float64
		bool   *bool
		text   *string
		json   *string
		jsonb  *string
		date   *time.Time
	}

	date, _ = time.Parse(time.RFC3339, "2022-10-11T19:00:00Z")
	ssp := StructPointers{
		lo.ToPtr(int16(30_000)),
		lo.ToPtr(int32(200_000)),
		lo.ToPtr(int64(5_000_000_000)),
		lo.ToPtr(float32(3.14)),
		lo.ToPtr(3.1415),
		lo.ToPtr(true),
		lo.ToPtr("smoothdb"),
		lo.ToPtr("[123]"),
		lo.ToPtr("[\"test\"]"),
		lo.ToPtr(date),
	}
	s = structs[0]

	if !compareStruct(s, ssp) {
		t.Errorf("Expected \n\t\"%v\", \ngot \n\t\"%v\" \n", ssp, s)
	}

	ssp = StructPointers{}
	s = structs[1]

	if !compareStruct(s, ssp) {
		t.Errorf("Expected \n\t\"%v\", \ngot \n\t\"%v\" \n", ssp, s)
	}

	ssp = StructPointers{
		lo.ToPtr(int16(0)),
		lo.ToPtr(int32(0)),
		lo.ToPtr(int64(0)),
		lo.ToPtr(float32(0)),
		lo.ToPtr(0.0),
		lo.ToPtr(false),
		lo.ToPtr(""),
		lo.ToPtr("{}"),
		lo.ToPtr("{}"),
		lo.ToPtr(time.Time{}),
	}
	s = structs[2]

	if !compareStruct(s, ssp) {
		t.Errorf("Expected \n\t\"%v\", \ngot \n\t\"%v\" \n", ssp, s)
	}
}
