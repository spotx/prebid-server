package spotx

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "appnexustest", NewAdapter(nil, "http://search.spotxchange.com/openrtb"))
}