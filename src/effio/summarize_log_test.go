package effio

import "testing"

type testBs struct {
	bins    int
	records int
	expect  int
}

var bsTestData = []testBs{
	{1, 1, 1},
	{1, 0, 0},
	{10, 100, 10},
	{10, 101, 10},
	{10, 150, 15},
	{10, 175, 17},
	{101, 175, 1},
	{100, 1750123, 17501},
}

func TestBucketSize(t *testing.T) {
	for _, tbs := range bsTestData {
		sz := bucketSize(tbs.bins, tbs.records)
		if sz != tbs.expect {
			t.Error("bucketSize(", tbs.bins, ",", tbs.records, ") should = ", tbs.expect, " but got ", sz)
		}
	}
}
