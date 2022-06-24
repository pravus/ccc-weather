package main

import (
	"strings"
	"testing"

	"github.com/antchfx/htmlquery"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	{
		doc, err := htmlquery.Parse(strings.NewReader(`
			<html>
				<head></head>
				<body>
					<div class="myforecast-current-lrg">123°F</div>
					<div class="myforecast-current-sm">-45°C</div>
				</body>
			</html>
		`))
		require.NoError(t, err)
		v, u, err := parse(doc, `myforecast-current-lrg`)
		require.NoError(t, err)
		require.Equal(t, 123.0, v)
		require.Equal(t, `F`, u)
		v, u, err = parse(doc, `myforecast-current-sm`)
		require.NoError(t, err)
		require.Equal(t, -45.0, v)
		require.Equal(t, `C`, u)
	}

	{
		doc, err := htmlquery.Parse(strings.NewReader(``))
		require.NoError(t, err)
		_, _, err = parse(doc, `foo`)
		require.ErrorContains(t, err, `no nodes found for "foo"`)
	}

	{
		doc, err := htmlquery.Parse(strings.NewReader(`<fake class="foo">987°K</fake>`))
		require.NoError(t, err)
		_, _, err = parse(doc, `foo`)
		require.ErrorContains(t, err, `malformed node found for "foo"`)
	}

	{
		doc, err := htmlquery.Parse(strings.NewReader(`<fake class="foo">°F</fake>`))
		require.NoError(t, err)
		_, _, err = parse(doc, `foo`)
		require.ErrorContains(t, err, `malformed node found for "foo"`)
	}

	{
		doc, err := htmlquery.Parse(strings.NewReader(`<fake class="foo">123 °F</fake>`))
		require.NoError(t, err)
		_, _, err = parse(doc, `foo`)
		require.ErrorContains(t, err, `malformed node found for "foo"`)
	}

	{
		doc, err := htmlquery.Parse(strings.NewReader(`<fake class="foo">123°f</fake>`))
		require.NoError(t, err)
		_, _, err = parse(doc, `foo`)
		require.ErrorContains(t, err, `malformed node found for "foo"`)
	}

	{
		doc, err := htmlquery.Parse(strings.NewReader(`<fake class="foo">99999999999999999999999999999999°F</fake>`))
		require.NoError(t, err)
		_, _, err = parse(doc, `foo`)
		require.ErrorContains(t, err, `strconv.ParseInt: parsing "99999999999999999999999999999999": value out of range`)
	}
}
