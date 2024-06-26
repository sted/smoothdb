package database

import (
	"encoding/binary"
	"math"
	"time"
)

const microsecFromUnixEpochToY2K int64 = 946684800 * 1000000
const secFromUnixEpochToY2K int64 = 946684800

func toInt16(buf []byte) int16 {
	return int16(binary.BigEndian.Uint16(buf))
}

func toInt32(buf []byte) int32 {
	return int32(binary.BigEndian.Uint32(buf))
}

func toInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

func toFloat32(buf []byte) float32 {
	n := binary.BigEndian.Uint32(buf)
	return math.Float32frombits(n)
}

func toFloat64(buf []byte) float64 {
	n := binary.BigEndian.Uint64(buf)
	return math.Float64frombits(n)
}

func toBool(buf []byte) bool {
	return buf[0] == 1
}

func toDate(buf []byte) time.Time {
	daysSinceY2K := int32(binary.BigEndian.Uint32(buf))
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(daysSinceY2K))
}

func toTimestamp(buf []byte) time.Time {
	microsecSinceY2K := int64(binary.BigEndian.Uint64(buf))
	return time.Unix(
		secFromUnixEpochToY2K+microsecSinceY2K/1_000_000,
		(microsecFromUnixEpochToY2K+microsecSinceY2K)%1_000_000*1_000).UTC()
}
