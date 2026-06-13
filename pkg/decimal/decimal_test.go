package decimal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fathimasithara01/multitrade-platform/pkg/decimal"
)

func TestParsePositive(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		{"100.00", false},
		{"0.00000001", false},
		{"0", true},
		{"-1", true},
		{"abc", true},
		{"", true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			_, err := decimal.ParsePositive(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAddSubMul(t *testing.T) {
	a, _ := decimal.Parse("100.00000000")
	b, _ := decimal.Parse("50.00000000")

	assert.Equal(t, "150.00000000", decimal.Format(decimal.Add(a, b)))
	assert.Equal(t, "50.00000000", decimal.Format(decimal.Sub(a, b)))
	assert.Equal(t, "5000.00000000", decimal.Format(decimal.Mul(a, b)))
}

func TestMin(t *testing.T) {
	a, _ := decimal.Parse("3.00000000")
	b, _ := decimal.Parse("7.00000000")

	assert.Equal(t, "3.00000000", decimal.Format(decimal.Min(a, b)))
	assert.Equal(t, "3.00000000", decimal.Format(decimal.Min(b, a)))
}

func TestFormat(t *testing.T) {
	f, err := decimal.Parse("1.5")
	require.NoError(t, err)
	assert.Equal(t, "1.50000000", decimal.Format(f))
}

func TestMustParse_ValidInput(t *testing.T) {
	f := decimal.MustParse("999.12345678")
	assert.Equal(t, "999.12345678", decimal.Format(f))
}
