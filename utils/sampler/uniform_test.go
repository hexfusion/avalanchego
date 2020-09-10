// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sampler

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanche-go/utils"
)

var (
	uniformSamplers = []struct {
		name    string
		sampler Uniform
	}{
		{
			name:    "replacer",
			sampler: &uniformReplacer{},
		},
		{
			name:    "resampler",
			sampler: &uniformResample{},
		},
		{
			name:    "best",
			sampler: NewBestUniform(30),
		},
	}
	uniformTests = []struct {
		name string
		test func(*testing.T, Uniform)
	}{
		{
			name: "initialize overflow",
			test: UniformInitializeOverflowTest,
		},
		{
			name: "out of range",
			test: UniformOutOfRangeTest,
		},
		{
			name: "empty",
			test: UniformEmptyTest,
		},
		{
			name: "singleton",
			test: UniformSingletonTest,
		},
		{
			name: "distribution",
			test: UniformDistributionTest,
		},
		{
			name: "over sample",
			test: UniformOverSampleTest,
		},
	}
)

func TestAllUniform(t *testing.T) {
	for _, s := range uniformSamplers {
		for _, test := range uniformTests {
			t.Run(fmt.Sprintf("sampler %s test %s", s.name, test.name), func(t *testing.T) {
				test.test(t, s.sampler)
			})
		}
	}
}

func UniformInitializeOverflowTest(t *testing.T, s Uniform) {
	err := s.Initialize(math.MaxUint64)
	assert.Error(t, err, "should have reported an overflow error")
}

func UniformOutOfRangeTest(t *testing.T, s Uniform) {
	err := s.Initialize(0)
	assert.NoError(t, err)

	_, err = s.Sample(1)
	assert.Error(t, err, "should have reported an out of range error")
}

func UniformEmptyTest(t *testing.T, s Uniform) {
	err := s.Initialize(1)
	assert.NoError(t, err)

	val, err := s.Sample(0)
	assert.NoError(t, err)
	assert.Len(t, val, 0, "shouldn't have selected any element")
}

func UniformSingletonTest(t *testing.T, s Uniform) {
	err := s.Initialize(1)
	assert.NoError(t, err)

	val, err := s.Sample(1)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{0}, val, "should have selected the only element")
}

func UniformDistributionTest(t *testing.T, s Uniform) {
	err := s.Initialize(3)
	assert.NoError(t, err)

	val, err := s.Sample(3)
	assert.NoError(t, err)

	utils.SortUint64(val)
	assert.Equal(
		t,
		[]uint64{0, 1, 2},
		val,
		"should have selected the only element",
	)
}

func UniformOverSampleTest(t *testing.T, s Uniform) {
	err := s.Initialize(3)
	assert.NoError(t, err)

	_, err = s.Sample(4)
	assert.Error(t, err, "should have returned an out of range error")
}
